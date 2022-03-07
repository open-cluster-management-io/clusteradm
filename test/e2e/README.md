[comment]: # ( Copyright Contributors to the Open Cluster Management project )
# How to write e2e test for Clusteradm?

## TestE2eConfig
`TestE2eConfig` has 3 methods: 

```
Cluster()
Clusteradm() 
ResetEnv()
ClearEnv()
```

In the following contents, `e2e` is an instance of `TestE2eConfig`.

---
### Cluster()
You can get cluster informations by call Cluster(). Such as:
```
// name of hub
e2e.Cluster().Hub().Name()

// context of managedcluster1
e2e.Cluster().ManagedCluster1().Context()
``` 

we need to specify name/context in some of our commands.

---
### Clusteradm() 

How to run a command?
We just need to pass run flags to the corresponding command method.
the function call looks like:
```
err := e2e.Clusteradm().<Subcommand-Method>.Run()
```

`<Subcommand-Method>` can be one of  `Version()`, 
`Init(args ...string)`,
`Join(args ...string)` ,
`Accept(args ...string)` ,
`Get(args ...string)` ,
`Delete(args ...string)` ,
`Addon(args ...string)` ,
`Clean(args ...string)` ,
`Install(args ...string)` ,
`Proxy(args ...string)` ,
`Unjoin(args ...string)`.

---
### reset e2e environment

#### 1. ResetEnv
First, we define an initial e2e environment state: 1 hub, 2 managed(cluster1, cluster2), hub is initialized, cluster1 is join and accepted, cluster2 is just created.

You need to call `ResetEnv` method to reset the e2e environment to initial state after each test senario which made a change on the test environment.
```go
e2e.ResetEnv()
```

#### 2. ClearEnv
In some test senario, you may need an empty environment, you can call `ClearEnv` method before your test senario, it will help you to clear the resourses applied on cluster.
```go
e2e.ClearEnv()
``` 

---


## write test

Now we will show you how to write tests for clusteradm command.

0.  We are bootstrapping our clusters in `e2e-test.mk`, if you're not using [kind](https://kind.sigs.k8s.io/docs/user/quick-start/) to create clusters, please modify the create/delete commands and the context environment variables.

1.  First, what you need to do is: Call `e2e.ResetEnv()` in `ginkgo.AfterEach` to make sure the environment is reset to initial state after your test code have made changes to the environment,.

    And if you just want an empty environment, call `e2e.ClearEnv()` in  `ginkgo.AfterEach` to make sure the applied resources are removed before youe test cases start.


    *Actually, we've started the e2e environment for you, which includes 3 clusters(1 hub, 2 managed clusters). Hub is initialized and managed cluster1 is joined&accepted to hub.*



2.  Second, if you want to test the command `clusteradm version`. 
    ```
    err := e2e.Clusteradm().Version()
    ```
    The command will be executed, then you just need to focus on the err. 

    But if the output of a command such as `init` or `get token` needs to be reuse? 
    Don't worry, the output has been automatically resolved and stored in e2e instance. while you want to use this, just call:
    ```
    e2e.CommandResult().Token()
    e2e.CommandResult().Host()
    ```

    But, only the latest one whill be stored in e2e instance. Do store the result using `tmp := e2e.CommandResult()` if you need.

For more details, see other files in this directory.
