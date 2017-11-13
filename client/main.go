package main

import (
    "io"
    "log"

    "golang.org/x/net/context"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"

    pb "poc-grpc-protobuf-go/customer"
)

const (
    address = "localhost:50051"
)

// Authentication holds the login/password
type Authentication struct {
    Login    string
    Password string
}

// GetRequestMetadata gets the current request metadata
func (a *Authentication) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
    return map[string]string{
      "login":    a.Login,
      "password": a.Password,
    }, nil
}

// RequireTransportSecurity indicates whether the credentials requires transport security
func (a *Authentication) RequireTransportSecurity() bool {
    return true
}

// createCustomer calls the RPC method CreateCustomer of CustomerServer
func createCustomer(client pb.CustomerClient, customer *pb.CustomerRequest) {
    resp, err := client.CreateCustomer(context.Background(), customer)
    if err != nil {
        log.Fatalf("Could not create Customer: %v", err)
    }
    if resp.Success {
        log.Printf("A new Customer has been added with id: %d", resp.Id)
    }
}

// GetCustomers calls the RPC method GetCustomers of CustomerServer
func getCustomers(client pb.CustomerClient, filter *pb.CustomerFilter) {
    // calling the streaming API
    stream, err := client.GetCustomers(context.Background(), filter)
    if err != nil {
        log.Fatal("Error on get customers: %v", err)
    }
    for {
        customer, err := stream.Recv()
        if err == io.EOF {
            break
        }

        if err != nil {
            log.Fatal("%v.GetCustomers(_) = _, %v", client, err)
        }
        log.Printf("Customer: %v", customer)
    }
}

func main() {
    creds, err := credentials.NewClientTLSFromFile("../cert/server.crt", "")
    if err != nil {
      log.Fatalf("could not load tls cert: %s", err)
    }

    auth := Authentication{
        Login:    "john",
        Password: "doe",
    }

    // Set up a connection to the RPC server
    conn, err := grpc.Dial(address, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(&auth))
    if err != nil {
        log.Fatal("did not connect: %v", err)
    }
    defer conn.Close()
    // creates a new CustomerClient
    client := pb.NewCustomerClient(conn)

    customer := &pb.CustomerRequest{
        Id:    103,
        Name:  "Shiju Varghese",
        Email: "shiju@xyz.com",
        Phone: "732-757-2923",
        Addresses: []*pb.CustomerRequest_Address{
            &pb.CustomerRequest_Address{
                Street:            "1 Mission Street",
                City:              "San Francisco",
                State:             "CA",
                Zip:               "94105",
                IsShippingAddress: false,
            },
            &pb.CustomerRequest_Address{
                Street:            "Greenfield",
                City:              "Kochi",
                State:             "KL",
                Zip:               "68356",
                IsShippingAddress: true,
            },
        },
    }

    // Create a new customer
    createCustomer(client, customer)

    customer = &pb.CustomerRequest{
        Id:    102,
        Name:  "Irene Rose",
        Email: "irene@xyz.com",
        Phone: "732-757-2924",
        Addresses: []*pb.CustomerRequest_Address{
            &pb.CustomerRequest_Address{
                Street:            "1 Mission Street",
                City:              "San Francisco",
                State:             "CA",
                Zip:               "94105",
                IsShippingAddress: true,
            },
        },
    }

    // Create a new customer
    createCustomer(client, customer)
    //Filter with an empty Keyword
    filter := &pb.CustomerFilter{Keyword: ""}
    getCustomers(client, filter)

}