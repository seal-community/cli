# Seal CLI

The Seal CLI allows users to easily replace vulnerable packages in their projects with sealed, vulnerability-free versions, which are available for download from Seal's artifact server.

Currently the CLI supports `npm` projects, and projects that use `yarn v1`. Other package managers will be added over time.

We recommended you incorporate this tool as part of the CI process immediately after the packages are pulled from the artifact server (e.g `npm install`). If you're using GitHub Actions as your CI just use our [action](https://github.com/seal-community/cli-action).

## Usage

### Scanning
The scan phase does not require authentication.
However, it relies on the package managers to read the project's dependency tree and parse it.
Therefore, to get a complete result the dependencies must already be installed.

1. Go to the root directory of the project and install its dependencies (e.g `npm install`).
2. Run `seal scan`. To save the output as a CSV use `seal scan --csv output.txt`. The dependencies will be checked against several vulnerability databases (such as OSV).
3. The results will be presented as a table of packages and vulnerabilities, for example:

| LIBRARY       | VERSION | ECOSYSTEM | VULNERABILITIES                                | CAN SEAL | SEALED VERSION |
| :------------ | :------ | :-------- | :--------------------------------------------- | :------: | :------------- |
| d3-color      | 2.0.0   | Node      | GHSA-36jr-mh4h-2g58 (5.3)                      |    V     | 2.0.0-sp1      |
| semver        | 7.0.0   | Node      | CVE-2022-25883 (7.5)                           |    V     | 7.0.0-sp1      |
| set-value     | 3.0.3   | Node      | CVE-2021-23440 (7.3)                           |    X     |                |
| passport-saml | 1.5.0   | Node      | CVE-2022-39299 (8.1) <br> CVE-2021-39171 (5.3) |    V     | 1.5.0-sp1      |
| axios         | 0.21.4  | Node      | CVE-2023-45857 (7.1)                           |    V     | 0.21.4-sp1     |

The `CAN SEAL` and `SEALED VERSION` columns show whether the particular vulnerable package has a patched version that is available on Seal's artifact server.

### Fixing
To fix the vulnerabilities using the CLI you will need an access token to the sealed packages on the Seal artifact server.
You can register [here](https://app.sealsecurity.io/).

1. Go to the root directory of the project and install its dependencies (e.g `npm install`).
2. Set the access token and project ID. There are two ways to do this:
	1. Set the access token in the `SEAL_TOKEN` environment variable, and the project ID in `SEAL_PROJECT`.
	```bash
	export SEAL_TOKEN=ey534tj9htrmoikNMNDakn43jaI5453tjkthspj==
	```
	```bash
	export SEAL_PROJECT=my-test-project
	```
	2. Set the access token and project ID in the `.seal-config.yml` configuration file in the local work directory as in the following example:
	```yml
	token: ey534tj9htrmoikNMNDakn43jaI5453tjkthspj==
	project: my-test-project
	```
Note that the project ID is what uniquely identifies the project on the Seal platform.
It may only include ASCII letters, digits, underscores, hyphens or periods, and mustn't be over 255 characters long.

3. Run `seal fix --mode all`. All the vulnerable packages that have sealed versions will be replaced with them in place. If you prefer to fix only specific packages, see [here](#fixing-specific-packages).

Logging verbosity can be set by providing `-v`, `-vv` or `-vvv`.

### Fixing Specific Packages
The `seal fix --mode all` command fixes all the vulnerable packages for which a sealed version exists.
The CLI also supports selecting which packages to fix by using the ```seal fix --mode local``` command.
To specify which packages to seal use the `.seal-actions.yml` file (also known as the local configuration file).
The file is saved in the project's root directory, and contains instructions for the fix phase.

For each package you wish to fix use the following command: `seal add package-name version`.
For example: `seal add ejs 2.7.4`.
This command will add to the local configuration file an instruction to replace `ejs@2.7.4` with its sealed version `ejs@2.7.4-sp1`. If the file does not exist, a new one will be generated.<br/>
To generate a local configuration file with instructions to fix everything that has a sealed version in the project, you can use `seal scan --generate-local-config` instead of adding packages one-by-one.

Note that it's also possible to manually edit the `overrides` section in the local configuration file, as its format is straightforward:
```yml
    overrides:
	  ejs:
	    2.7.4:
		  use: 2.7.4-sp1
```

## How to Contribute
We're always looking for feedback, discuss possible integrations and receive feature requests.
Please open issues, pull requests, or contact us at [contribute@seal.security](mailto:contribute@seal.security).

## About Seal Security

![Seal Security Logo](docs/assets/logo.png)

Seal Security is an early-stage cybersecurity startup committed to simplifying vulnerability remediation for developers and application security practitioners. For more details, visit our [website](https://seal.security).
