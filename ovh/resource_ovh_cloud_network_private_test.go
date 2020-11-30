package ovh

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var testAccCloudNetworkPrivateConfig_attachVrack = `
resource "ovh_vrack_cloudproject" "attach" {
  service_name = "%s"
  project_id   = "%s"
}

data "ovh_cloud_regions" "regions" {
  service_name = ovh_vrack_cloudproject.attach.project_id

  has_services_up = ["network"]
}
`

var testAccCloudNetworkPrivateConfig_noAttachVrack = `
data "ovh_cloud_regions" "regions" {
  service_name = "%s"

  has_services_up = ["network"]
}
`

var testAccCloudNetworkPrivateConfig_basic = `
%s

resource "ovh_cloud_network_private" "network" {
  service_name = data.ovh_cloud_regions.regions.service_name
  vlan_id    = 0
  name       = "terraform_testacc_private_net"
  regions    = slice(sort(tolist(data.ovh_cloud_regions.regions.names)), 0, 3)
}
`

var testAccCloudNetworkPrivateDeprecatedConfig_basic = `
%s

resource "ovh_cloud_network_private" "network" {
  project_id = data.ovh_cloud_regions.regions.service_name
  vlan_id    = 0
  name       = "terraform_testacc_private_net"
  regions    = slice(sort(tolist(data.ovh_cloud_regions.regions.names)), 0, 3)
}
`

func testAccCloudNetworkPrivateConfig(config string) string {
	attachVrack := fmt.Sprintf(
		testAccCloudNetworkPrivateConfig_attachVrack,
		os.Getenv("OVH_VRACK_SERVICE_TEST"),
		os.Getenv("OVH_CLOUD_PROJECT_SERVICE_TEST"),
	)
	noAttachVrack := fmt.Sprintf(
		testAccCloudNetworkPrivateConfig_noAttachVrack,
		os.Getenv("OVH_CLOUD_PROJECT_SERVICE_TEST"),
	)

	if os.Getenv("OVH_ATTACH_VRACK") == "0" {
		return fmt.Sprintf(
			config,
			noAttachVrack,
		)
	}

	return fmt.Sprintf(
		config,
		attachVrack,
	)
}

func init() {
	resource.AddTestSweepers("ovh_cloud_network_private", &resource.Sweeper{
		Name: "ovh_cloud_network_private",
		F:    testSweepCloudNetworkPrivate,
	})
}

func testSweepCloudNetworkPrivate(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}

	vrack := os.Getenv("OVH_VRACK_SERVICE_TEST")
	if vrack == "" {
		log.Print("[DEBUG] OVH_VRACK_SERVICE_TEST is not set. No cloud_network_private to sweep")
		return nil
	}

	projectId := os.Getenv("OVH_CLOUD_PROJECT_SERVICE_TEST")
	if projectId == "" {
		log.Print("[DEBUG] OVH_CLOUD_PROJECT_SERVICE_TEST is not set. No cloud_network_private to sweep")
		return nil
	}

	networks := []CloudNetworkPrivateResponse{}
	err = client.Get(fmt.Sprintf("/cloud/project/%s/network/private", projectId), &networks)
	if err != nil {
		return fmt.Errorf("error listing private networks for project %q:\n\t %q", projectId, err)
	}

	for _, n := range networks {
		if !strings.HasPrefix(n.Name, test_prefix) {
			continue
		}

		log.Printf("[DEBUG] found dangling network & subnets for project: %s, id: %s", projectId, n.Id)
		err = resource.Retry(5*time.Minute, func() *resource.RetryError {
			subnetIds := []string{}
			err = client.Get(fmt.Sprintf("/cloud/project/%s/network/private/%s/subnet", projectId, n.Id), &subnetIds)
			if err != nil {
				return resource.RetryableError(fmt.Errorf("error listing private network subnets for project %q:\n\t %q", projectId, err))
			}

			for _, s := range subnetIds {
				if err := client.Delete(fmt.Sprintf("/cloud/project/%s/network/private/%s/subnet/%s", projectId, n.Id, s), nil); err != nil {
					return resource.RetryableError(err)
				}
			}

			if err := client.Delete(fmt.Sprintf("/cloud/project/%s/network/private/%s", projectId, n.Id), nil); err != nil {
				return resource.RetryableError(err)
			}

			// Successful cascade delete
			log.Printf("[DEBUG] successful cascade delete of network & subnets for project: %s, id: %s", projectId, n.Id)
			return nil
		})

		if err != nil {
			return err
		}

	}

	return nil
}

func TestAccCloudNetworkPrivate_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccCheckCloudNetworkPrivatePreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCloudNetworkPrivateConfig(testAccCloudNetworkPrivateConfig_basic),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("ovh_cloud_network_private.network", "service_name"),
					resource.TestCheckResourceAttrSet("ovh_cloud_network_private.network", "id"),
					resource.TestCheckResourceAttr("ovh_cloud_network_private.network", "vlan_id", "0"),
				),
			},
		},
	})
}

func TestAccCloudNetworkPrivateDeprecated_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccCheckCloudNetworkPrivatePreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCloudNetworkPrivateConfig(testAccCloudNetworkPrivateDeprecatedConfig_basic),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("ovh_cloud_network_private.network", "service_name"),
					resource.TestCheckResourceAttrSet("ovh_cloud_network_private.network", "id"),
					resource.TestCheckResourceAttr("ovh_cloud_network_private.network", "vlan_id", "0"),
				),
			},
		},
	})
}

func testAccCheckCloudNetworkPrivatePreCheck(t *testing.T) {
	testAccPreCheckCloud(t)
	testAccCheckCloudExists(t)
	testAccPreCheckVRack(t)
}
