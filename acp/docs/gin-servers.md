### Working with Gin servers

Gin provides a powerful context object that be used to read request information and send down response information.

For example

```
c.Request.Context()
```

```
c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
```

However, gin workflows can be a bit hard to read, and can lead to subtle bugs around 
when to `return` after calling methods on `c` that generate responses.

To mitigate this, all gin endpoint should be short and encapsulate all
logic that relies on the gin context.

The following method is bad for a few reasons:

- need to remember to `return` after calling methods on `c` that generate responses
- hard to tell from the signature what the method's input/outputs are
- duplicated error handling code makes it easy to sprawl different handling logic
- side-effecting functions (methods on `c`) are woven throughout what could be a mostly-pure (and therefore easy to test) function

```go
func (s *Server) createUser(c *gin.Context) {
    var user User
    if err := c.ShouldBindJSON(&user); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    // do stuff with the user inline
    // ... many lines of code
    if someErrorCondition {
        c.JSON(http.StatusBadRequest, gin.H{"error": "some error"})
        return
    }
    // ... many more lines of code
    if someOtherErrorCondition {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "some other error"})
        return
    }
    // many more lines of code
    c.JSON(http.StatusOK, gin.H{"message": "User created successfully"})
}
```

you should ALWAYS move business logic out of gin handlers and into a separate function with a clean type signature. Methods that handle business logic should not know about gin or http.

```go
func createUser(ctx context.Context, user User) (*User, error) {
    // ... do stuff with user
    // many lines of code
    if someErrorCondition {
        return nil, errors.New("some error")
    }
    // ... many more lines of code
    if someOtherErrorCondition {
        return nil, errors.New("some other error")
    }
    // ... many more lines of code
    return &user, nil
}
func (s *Server) createUserHandler(c *gin.Context) {
    var user User
    // validate in gin handler
    if err := c.ShouldBindJSON(&user); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    // business logic doesn't know about gin
    user, err := createUser(c.Request.Context(), user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, user)
}
```

this will lead to a more readable and maintainable function. Benefits include

- decouples important business logic from gin interfaces
- can switch off gin or use business logic in non-gin contexts (e.g. CLI, queue workers, etc)
- methods can return a standard error type, without having gin-specific error handling code 
    - (can still define a custom type like ErrInvalid and map that to error codes like 4xx, 5xx, etc in the gin handler)
- type signatures in biz logic make it easy to understand what is happening
- easier to test business logic without having to mock gin or test gin internals
