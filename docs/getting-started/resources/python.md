---
outline: deep
---

# Python Resource

The `python` resource is designed to run Python scripts specified in the `script` attribute from the workflow.

This resource also provides access to installed Python packages installed via `pip`, and as well as
[Anaconda](https://www.anaconda.com) packages defined in the `workflow.pkl`, are available for use. The resource also
includes a `condaEnvironment` attribute, allowing it to isolate execution within a specified Conda environment. For more
information, see the [Workflow](../configuration/workflow) documentation.

## Creating a New Python Resource

To create a new `python` resource, you can either generate a new AI agent using the `kdeps new` command or `scaffold`
the resource directly.

Here’s how to scaffold a `python` resource:

```bash
kdeps scaffold [aiagent] python
```

This command will add a `python` resource into the `aiagent/resources` folder, generating the following folder structure:

```text
aiagent
└── resources
    └── python.pkl
```

The file includes essential metadata and common configurations, such as [Skip Conditions](../resources/skip) and
[Preflight Validations](../resources/validations) settings. For more details, refer to the [Common Resource
Configurations](../resources/resources#common-resource-configurations) documentation.

## Python Block

Within the file, you’ll find the `python` block, which is structured as follows:

```apl
python {
    script = """
    print("hello world")
    """
    env {
        // Environment variables accessible within the script
        ["ENVVAR"] = "XYZ"  // Example environment variable
    }
    // Specifies the timeout duration (in seconds) for script execution
    timeoutSeconds = 60

    // Specifies the Conda environment for isolation
    condaEnvironment = "my-conda-env"
}
```

Key elements of the `python` block include:

- **`script`**: Specifies the Python script to execute, enclosed in triple double-quotes (`"""`) for multi-line support.
- **`env`**: Defines environment variables to be available during execution.
- **`timeoutSeconds`**: Determines the execution timeout in seconds, after which the script execution will be terminated.
- **`condaEnvironment`**: Specifies the Conda environment to use, ensuring the script runs in an isolated environment
  with defined dependencies.

When the resource is executed, you can leverage Python functions like `python.stdout("id")` to access the output. For
further details, refer to the [Python Functions](../resources/functions.md#python-resource-functions) documentation.
