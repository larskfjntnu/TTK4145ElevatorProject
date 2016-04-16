#!/bin/bash
clear

echo "Enter sanntid username:"
read user
echo "Enter last byte of the starting elevators IP:"
read IP
echo "Connecting to 129.241.187."$IP
scp -rq Main $user@129.241.187.$IP:~/gr66
ssh $user@129.241.187.$IP

# Run this in ssh:
	# cd gr66/
	#go run Main.go