package main

import (
    "log"
    "net"
    "strings"

    "golang.org/x/net/context"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"

    pb "poc-grpc-protobuf-go/customer"    
)

const (
    port = ":50051"
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

func main() {
    lis, err := net.Listen("tcp", port)
    if err != nil {
        log.Fatal("failed to listen: %v", err)
    }

    creds, err := credentials.NewServerTLSFromFile("../cert/server.crt", "../cert/server.key")
    if err != nil {
      log.Fatalf("could not load TLS keys: %s", err)
    }

    opts := []grpc.ServerOption{grpc.Creds(creds)}

    s := grpc.NewServer(opts...)

    pb.RegisterCustomerServer(s, &server{})
    
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %s", err)
    }
}