# KeyloGo

This project is a simple implementation of a Keylogger using golang.

The idea behind it is purely to get a better understand about how keylogger works.

## Linux

The linux implementation gets all connected devices (excepting the ones that contain mouse in the name) and start writing a file translating the input.

This approach brought an issue about having a lot of wrong devices lists, so I used a function that checks the usage of each device and if there's no input after a few time it will clean the entries generated for it and stop listening to this device.

*Remember that sudo is necessary for running it*

## Windows

Must be implemented. 

## Goal

The idea is to autorun it on any external source (Eg:. Pendrive's) and found a way to escalate permissions, also using a shell to select which binary it should execute based on the OS.

## TODO

 - [X] Linux implementation
 - [X] Garbage collector for unused devices
 - [ ] Sending data to server automatically
 - [ ] Windows implementation
 - [ ] Privileges escalation
 - [ ] External source Autorun