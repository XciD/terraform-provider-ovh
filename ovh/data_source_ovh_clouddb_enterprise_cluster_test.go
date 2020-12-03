package ovh

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccCloudDBEnterpriseCluster(t *testing.T) {
	clusterId := os.Getenv("OVH_CLOUDDB_ENTERPRISE")
	config := fmt.Sprintf(testAccCloudDBEnterpriseClusterDatasourceConfig, clusterId)
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckCloudDBEnterpriseCluster(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.ovh_clouddb_enterprise_cluster.db", "cluster_id", clusterId),
					resource.TestCheckResourceAttr(
						"data.ovh_clouddb_enterprise_cluster.db", "status", string(CloudDBEnterpriseClusterStatusCreated)),
				),
			},
		},
	})
}

const testAccCloudDBEnterpriseClusterDatasourceConfig = `
data "ovh_clouddb_enterprise_cluster" "db" {
  cluster_id = "%s"
}
`
