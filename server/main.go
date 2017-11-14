package main

import (
    "fmt"
    "log"
    "net"
    "net/http"    
    "strings"

    "github.com/grpc-ecosystem/grpc-gateway/runtime"
    "golang.org/x/net/context"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
    "google.golang.org/grpc/metadata"    

    pb "poc-grpc-protobuf-go/customer"
)

type contextKey int

const (
    port = ":50051"
    clientIDKey contextKey = iota
)

// server is used to implement customer.CustomerServer.
type server struct {
    savedCustomers []*pb.CustomerRequest
}

// CreateCustomer creates a new Customer
func (s *server) CreateCustomer(ctx context.Context, in *pb.CustomerRequest) (*pb.CustomerResponse, error) {
    s.savedCustomers = append(s.savedCustomers, in)
    return &pb.CustomerResponse{Id: in.Id, Success: true}, nil
}

// GetCustomers returns all customers by given filter
func (s *server) GetCustomers(filter *pb.CustomerFilter, stream pb.Customer_GetCustomersServer) error {
    for _, customer := range s.savedCustomers {
        if filter.Keyword != "" {
            if !strings.Contains(customer.Name, filter.Keyword) {
                continue
            }
        }
        if err := stream.Send(customer); err != nil {
            return err
        }
    }
    return nil
}

func credMatcher(headerName string) (mdName string, ok bool) {
    if headerName == "Login" || headerName == "Password" {
        return headerName, true
    }
    return "", false
}

func authenticateClient(ctx context.Context, s *server) (string, error) {
    if md, ok := metadata.FromIncomingContext(ctx); ok {
      clientLogin := strings.Join(md["login"], "")
      clientPassword := strings.Join(md["password"], "")

      if clientLogin != "john" {
            return "", fmt.Errorf("unknown user %s", clientLogin)
      }

      if clientPassword != "doe" {
            return "", fmt.Errorf("bad password %s", clientPassword)
      }

      log.Printf("authenticated client: %s", clientLogin)
      return "42", nil
    }

    return "", fmt.Errorf("missing credentials")
}

func unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    s, ok := info.Server.(*server)
    if !ok {
        return nil, fmt.Errorf("unable to cast server")
    }
    
    clientID, err := authenticateClient(ctx, s)
    if err != nil {
        return nil, err
    }

    ctx = context.WithValue(ctx, clientIDKey, clientID)
    return handler(ctx, req)
}

func startGRPCServer(address, certFile, keyFile string) error {
    lis, err := net.Listen("tcp", address)
    if err != nil {
        return fmt.Errorf("failed to listen: %v", err)
    }

    creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
    if err != nil {
        return fmt.Errorf("could not load TLS keys: %s", err)
    }

    opts := []grpc.ServerOption{grpc.Creds(creds), grpc.UnaryInterceptor(unaryInterceptor)}

    s := grpc.NewServer(opts...)

    pb.RegisterCustomerServer(s, &server{})
    
    if err := s.Serve(lis); err != nil {
        return fmt.Errorf("failed to serve: %s", err)
    }

    return nil
}

func startRESTServer(address, grpcAddress, certFile string) error {
    ctx := context.Background()
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()

    mux := runtime.NewServeMux(runtime.WithIncomingHeaderMatcher(credMatcher))
    creds, err := credentials.NewClientTLSFromFile(certFile, "")
    if err != nil {
        return fmt.Errorf("could not load TLS certificate: %s", err)
    }

    opts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}

    err = pb.RegisterCustomerHandlerFromEndpoint(ctx, mux, grpcAddress, opts)
    if err != nil {
        return fmt.Errorf("could not register service Ping: %s", err)
    }

    log.Printf("starting HTTP/1.1 REST server on %s", address)

    http.ListenAndServe(address, mux)

    return nil
}

func main() {
    grpcAddress := fmt.Sprintf("%s:%d", "localhost", 7777)
    restAddress := fmt.Sprintf("%s:%d", "localhost", 7778)
    certFile := "cert/server.crt"
    keyFile := "cert/server.key"

    go func() {
        err := startGRPCServer(grpcAddress, certFile, keyFile)
        if err != nil {
            log.Fatalf("failed to start gRPC server: %s", err)
        }
    }()

    go func() {
        err := startRESTServer(restAddress, grpcAddress, certFile)
        if err != nil {
            log.Fatalf("failed to start gRPC server: %s", err)
        }
    }()

    log.Printf("Entering infinite loop")
    select {}
}