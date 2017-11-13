package main

import (
    "fmt"
    "log"
    "net"
    "strings"

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

func main() {
    lis, err := net.Listen("tcp", port)
    if err != nil {
        log.Fatal("failed to listen: %v", err)
    }

    creds, err := credentials.NewServerTLSFromFile("../cert/server.crt", "../cert/server.key")
    if err != nil {
      log.Fatalf("could not load TLS keys: %s", err)
    }

    opts := []grpc.ServerOption{grpc.Creds(creds), grpc.UnaryInterceptor(unaryInterceptor)}

    s := grpc.NewServer(opts...)

    pb.RegisterCustomerServer(s, &server{})
    
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %s", err)
    }
}