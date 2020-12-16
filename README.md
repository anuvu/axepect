# Some expect related things for cimc and login shell.

Most of the function here can be gleaned from cimc/cmd/main.go.

Things you can do.
 * connect:

     ```go
	sess, err := cimc.Session("admin", "SUPER-SECRET", "10.0.1.1")
    ```

 * send a command, get the reply.

     ```go
     resp, err := sess.SendCmd("show cimc")
     ```

 * handle scope commands transparently.  If the first char of the command is a '/'
     then the command will be run in the provided scope.

     ```go
     resp, err := sess.SendCmd("/chassis/power on")
     ```

     will send:

         * `top`
         * `scope /chassis`
         * power on

 * connect to the console, use [goexpect](https://github.com/google/goexpect) on your own, then go back to cimc shell.

     ```golang
     e, err := sess.ExpectHostConsole()

     e.Send("username\n")
     sess.EndHostConsole()

     ```

## Build
Type `make`
