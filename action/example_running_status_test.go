package action_test

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/cnabio/cnab-go/action"
	"github.com/cnabio/cnab-go/claim"
	"github.com/cnabio/cnab-go/credentials"
	"github.com/cnabio/cnab-go/driver/lookup"
	"github.com/cnabio/cnab-go/utils/crud"
)

// appendFailedResult creates a failed result from the operation error and accumulates
// the error(s).
func appendFailedResult(opErr error, cp claim.Provider, c claim.Claim) error {
	saveResult := func() error {
		result, err := c.NewResult(claim.StatusFailed)
		if err != nil {
			return err
		}
		return cp.SaveResult(result)
	}

	resultErr := saveResult()

	// Accumulate any errors from the operation with the persistence errors
	return multierror.Append(opErr, resultErr).ErrorOrNil()
}

// Upgrade the bundle and record the operation as running immediately so that you can
// track how long the operation took.
func Example_runningStatus() {
	// Use the debug driver to only print debug information about the bundle but not actually execute it
	// Use "docker" to execute it for real
	d, err := lookup.Lookup("debug")
	if err != nil {
		panic(err)
	}

	// Use an in-memory store with no encryption
	cp := claim.NewClaimStore(crud.NewMockStore(), nil, nil)

	// Save an existing claim for the install action which has already taken place
	// This sets up data for us to use during our upgrade example
	createInstallClaim(cp)

	// Pass an empty set of parameters
	var parameters map[string]interface{}

	// Load the previous claim for the installation
	existingClaim, err := cp.ReadLastClaim("hello")
	if err != nil {
		panic(err)
	}

	// Create a claim for the upgrade operation based on the previous claim
	c, err := existingClaim.NewClaim(claim.ActionUpgrade, existingClaim.Bundle, parameters)
	if err != nil {
		panic(err)
	}

	// Set to consistent values so we can compare output reliably
	c.ID = "claim-id"
	c.Revision = "claim-rev"
	c.Created = time.Date(2020, time.April, 18, 1, 2, 3, 4, time.UTC)

	// Create the action that will execute the operation
	a := action.New(d, cp)

	// Persist every output generated by the bundle to our store
	a.SaveAllOutputs = true

	// Pass an empty set of credentials
	var creds credentials.Set

	// Save the upgrade claim in the Running Status
	err = a.SaveInitialClaim(c, claim.StatusRunning)
	if err != nil {
		panic(err)
	}

	opResult, claimResult, err := a.Run(c, creds)
	if err != nil {
		// If the bundle isn't run due to an error preparing
		// record a failure so we aren't left stuck in running
		err = appendFailedResult(err, cp, c)
		panic(err)
	}

	err = a.SaveOperationResult(opResult, c, claimResult)
	if err != nil {
		panic(err)
	}

	results, err := cp.ListResults(c.ID)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Created %d claim results\n", len(results))

	// Output: {
	//   "installation_name": "hello",
	//   "revision": "claim-rev",
	//   "action": "upgrade",
	//   "parameters": null,
	//   "image": {
	//     "imageType": "docker",
	//     "image": "example.com/myorg/myinstaller",
	//     "contentDigest": "sha256:7cc0618539fe11e801ce68911a0c9441a3dfaa9ba63057526c4016cf9db19474"
	//   },
	//   "environment": {
	//     "CNAB_ACTION": "upgrade",
	//     "CNAB_BUNDLE_NAME": "mybuns",
	//     "CNAB_BUNDLE_VERSION": "1.0.0",
	//     "CNAB_CLAIMS_VERSION": "1.0.0-DRAFT+b5ed2f3",
	//     "CNAB_INSTALLATION_NAME": "hello",
	//     "CNAB_REVISION": "claim-rev"
	//   },
	//   "files": {
	//     "/cnab/app/image-map.json": "{}",
	//     "/cnab/bundle.json": "{\"schemaVersion\":\"1.0.1\",\"name\":\"mybuns\",\"version\":\"1.0.0\",\"description\":\"\",\"invocationImages\":[{\"imageType\":\"docker\",\"image\":\"example.com/myorg/myinstaller\",\"contentDigest\":\"sha256:7cc0618539fe11e801ce68911a0c9441a3dfaa9ba63057526c4016cf9db19474\"}],\"actions\":{\"logs\":{}}}",
	//     "/cnab/claim.json": "{\"schemaVersion\":\"1.0.0-DRAFT+b5ed2f3\",\"id\":\"claim-id\",\"installation\":\"hello\",\"revision\":\"claim-rev\",\"created\":\"2020-04-18T01:02:03.000000004Z\",\"action\":\"upgrade\",\"bundle\":{\"schemaVersion\":\"1.0.1\",\"name\":\"mybuns\",\"version\":\"1.0.0\",\"description\":\"\",\"invocationImages\":[{\"imageType\":\"docker\",\"image\":\"example.com/myorg/myinstaller\",\"contentDigest\":\"sha256:7cc0618539fe11e801ce68911a0c9441a3dfaa9ba63057526c4016cf9db19474\"}],\"actions\":{\"logs\":{}}}}"
	//   },
	//   "outputs": {},
	//   "Bundle": {
	//     "schemaVersion": "1.0.1",
	//     "name": "mybuns",
	//     "version": "1.0.0",
	//     "description": "",
	//     "invocationImages": [
	//       {
	//         "imageType": "docker",
	//         "image": "example.com/myorg/myinstaller",
	//         "contentDigest": "sha256:7cc0618539fe11e801ce68911a0c9441a3dfaa9ba63057526c4016cf9db19474"
	//       }
	//     ],
	//     "actions": {
	//       "logs": {}
	//     }
	//   }
	// }
	// Created 2 claim results
}