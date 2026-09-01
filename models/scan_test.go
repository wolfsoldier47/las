package models

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestCallbackPayloadUnmarshal_AnsibleFormat(t *testing.T) {
	payloadJSON := `{
		"machine_name": "test123.zit.commerzbank.com",
		"os_type": "rhel",
		"os_verion": "9.7",
		"os_name": "RHEL",
		"stage": "ENTW",
		"datacentre": "Cloud{gcp,vmwareoncloud,azure}/vmware",
		"passwd_file": "[{\"dnsmasq\":\"x:980:980:Dnsmasq DHCP and DNS server:/var/lib/dnsmasq:/sbin/nologin\"},{\"tenabletag\":\"x:979:979::/opt/nessus_agent/var/nessus/mod/com.tenable.agent_identifier_service/data:/bin/false\"}]",
		"group_file": "[{\"gluster\":\"x:896:sam,sam2,sam3\"}]"
	}`

	var payload CallbackPayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if payload.MachineName != "test123.zit.commerzbank.com" {
		t.Errorf("expected machine_name test123.zit.commerzbank.com, got %s", payload.MachineName)
	}
	if payload.MachineType != OSTypeLinux {
		t.Errorf("expected machine_type linux, got %s", payload.MachineType)
	}
	if payload.OSVersion != "9.7" {
		t.Errorf("expected os_version 9.7, got %s", payload.OSVersion)
	}
	if payload.OSName != "RHEL" {
		t.Errorf("expected os_name RHEL, got %s", payload.OSName)
	}
	if payload.Environment != "ENTW" {
		t.Errorf("expected environment ENTW, got %s", payload.Environment)
	}
	if payload.Datacenter != "Cloud{gcp,vmwareoncloud,azure}/vmware" {
		t.Errorf("expected datacenter Cloud{gcp,vmwareoncloud,azure}/vmware, got %s", payload.Datacenter)
	}
	if len(payload.PasswdFile) != 2 {
		t.Errorf("expected 2 passwd entries, got %d", len(payload.PasswdFile))
	}
	if len(payload.GroupFile) != 1 {
		t.Errorf("expected 1 group entry, got %d", len(payload.GroupFile))
	}
	if payload.PasswdFile[0]["dnsmasq"] != "x:980:980:Dnsmasq DHCP and DNS server:/var/lib/dnsmasq:/sbin/nologin" {
		t.Errorf("unexpected passwd entry: %v", payload.PasswdFile[0])
	}
}

func TestCallbackPayloadUnmarshal_ArrayFileFields(t *testing.T) {
	payloadJSON := `{
		"machine_name": "host001",
		"machine_type": "solaris",
		"environment": "PROD",
		"DataCenter": "cloud",
		"passwd_file": [{"root": "x:0:0:root:/root:/bin/bash"}],
		"group_file": [{"root": "x:0:"}]
	}`

	var payload CallbackPayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if payload.MachineType != OSTypeSolaris {
		t.Errorf("expected solaris, got %s", payload.MachineType)
	}
	if len(payload.PasswdFile) != 1 {
		t.Errorf("expected 1 passwd entry, got %d", len(payload.PasswdFile))
	}
}

func TestCallbackPayloadUnmarshal_ArrayFormatWithDatacenter(t *testing.T) {
	payloadJSON := `{
		"machine_name": "test123.zit.commerzbank.com",
		"os_type": "solaris",
		"os_verion": "9.7",
		"os_name": "RHEL",
		"stage": "ENTW",
		"datacenter": "Cloud{gcp,vmwareoncloud,azure}/vmware",
		"passwd_file": [{"dnsmasq":"x:980:980:Dnsmasq DHCP and DNS server:/var/lib/dnsmasq:/sbin/nologin"},{"tenabletag":"x:979:979::/opt/nessus_agent/var/nessus/mod/com.tenable.agent_identifier_service/data:/bin/false"}],
		"group_file": [{"gluster":"x:896:sam,sam2,sam3"}]
	}`

	var payload CallbackPayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if payload.MachineType != OSTypeSolaris {
		t.Errorf("expected solaris, got %s", payload.MachineType)
	}
	if payload.OSVersion != "9.7" {
		t.Errorf("expected os_version 9.7, got %s", payload.OSVersion)
	}
	if payload.Environment != "ENTW" {
		t.Errorf("expected environment ENTW, got %s", payload.Environment)
	}
	if payload.Datacenter != "Cloud{gcp,vmwareoncloud,azure}/vmware" {
		t.Errorf("unexpected datacenter: %s", payload.Datacenter)
	}
	if len(payload.PasswdFile) != 2 {
		t.Errorf("expected 2 passwd entries, got %d", len(payload.PasswdFile))
	}
	if len(payload.GroupFile) != 1 {
		t.Errorf("expected 1 group entry, got %d", len(payload.GroupFile))
	}
	if payload.PasswdFile[0]["dnsmasq"] != "x:980:980:Dnsmasq DHCP and DNS server:/var/lib/dnsmasq:/sbin/nologin" {
		t.Errorf("unexpected passwd entry: %v", payload.PasswdFile[0])
	}
}

func TestCallbackEnvelopeUnmarshal_SingularHostKey(t *testing.T) {
	envelopeJSON := `{
		"ansible_job_id": "246985",
		"host": [
			{
				"machine_name": "uvdcp06.zit.commerzbank.com",
				"os_type": "Linux",
				"os_version": "8.10",
				"os_name": "RedHat",
				"stage": "prod",
				"datacentre": "ffm",
				"passwd_file": [{"root": "x:0:0:System Administrator of uvdcp06:/root:/bin/bash"}],
				"group_file": [{"root": "x:0:"}]
			}
		]
	}`

	var envelope CallbackEnvelope
	if err := json.Unmarshal([]byte(envelopeJSON), &envelope); err != nil {
		t.Fatalf("unmarshal envelope failed: %v", err)
	}

	if envelope.AnsibleJobID == nil {
		t.Fatal("expected ansible_job_id")
	}
	if fmt.Sprintf("%v", envelope.AnsibleJobID) != "246985" {
		t.Errorf("unexpected ansible_job_id: %v", envelope.AnsibleJobID)
	}
	if len(envelope.Hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(envelope.Hosts))
	}
	host := envelope.Hosts[0]
	if host.MachineName != "uvdcp06.zit.commerzbank.com" {
		t.Errorf("unexpected machine_name: %s", host.MachineName)
	}
	if host.MachineType != OSTypeLinux {
		t.Errorf("expected linux, got %s", host.MachineType)
	}
	if host.OSVersion != "8.10" {
		t.Errorf("expected os_version 8.10, got %s", host.OSVersion)
	}
	if host.Environment != "prod" {
		t.Errorf("expected environment prod, got %s", host.Environment)
	}
	if host.Datacenter != "ffm" {
		t.Errorf("expected datacenter ffm, got %s", host.Datacenter)
	}
	if len(host.PasswdFile) != 1 {
		t.Errorf("expected 1 passwd entry, got %d", len(host.PasswdFile))
	}
}

func TestCallbackEnvelopeUnmarshal(t *testing.T) {
	envelopeJSON := `{
		"ansible_job_id": 12345,
		"hosts": [
			{
				"machine_name": "test123.zit.commerzbank.com",
				"os_type": "rhel",
				"os_verion": "9.7",
				"os_name": "RHEL",
				"stage": "ENTW",
				"datacentre": "Cloud{gcp,vmwareoncloud,azure}/vmware",
				"passwd_file": "[{\"dnsmasq\":\"x:980:980:Dnsmasq DHCP and DNS server:/var/lib/dnsmasq:/sbin/nologin\"},{\"tenabletag\":\"x:979:979::/opt/nessus_agent/var/nessus/mod/com.tenable.agent_identifier_service/data:/bin/false\"}]",
				"group_file": "[{\"gluster\":\"x:896:sam,sam2,sam3\"}]"
			},
			{
				"machine_name": "test13.zit.commerzbank.com",
				"os_type": "solaris",
				"os_verion": "11.4",
				"stage": "TUC",
				"datacenter": "vmware",
				"passwd_file": [{"root": "x:0:0:root:/root:/bin/bash"}],
				"group_file": [{"root": "x:0:"}]
			}
		]
	}`

	var envelope CallbackEnvelope
	if err := json.Unmarshal([]byte(envelopeJSON), &envelope); err != nil {
		t.Fatalf("unmarshal envelope failed: %v", err)
	}

	if envelope.AnsibleJobID == nil {
		t.Fatal("expected ansible_job_id")
	}
	if fmt.Sprintf("%v", envelope.AnsibleJobID) != "12345" {
		t.Errorf("unexpected ansible_job_id: %v", envelope.AnsibleJobID)
	}
	if len(envelope.Hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(envelope.Hosts))
	}
	if envelope.Hosts[0].MachineName != "test123.zit.commerzbank.com" {
		t.Errorf("unexpected first host: %s", envelope.Hosts[0].MachineName)
	}
	if envelope.Hosts[0].MachineType != OSTypeLinux {
		t.Errorf("expected first host os_type linux, got %s", envelope.Hosts[0].MachineType)
	}
	if envelope.Hosts[1].MachineType != OSTypeSolaris {
		t.Errorf("expected second host os_type solaris, got %s", envelope.Hosts[1].MachineType)
	}
	if len(envelope.Hosts[0].PasswdFile) != 2 {
		t.Errorf("expected 2 passwd entries for first host, got %d", len(envelope.Hosts[0].PasswdFile))
	}
	if len(envelope.Hosts[1].PasswdFile) != 1 {
		t.Errorf("expected 1 passwd entry for second host, got %d", len(envelope.Hosts[1].PasswdFile))
	}
}

func TestOSTypeNormalization(t *testing.T) {
	tests := []struct {
		input    string
		expected OSType
	}{
		{"linux", OSTypeLinux},
		{"Linux", OSTypeLinux},
		{"LINUX", OSTypeLinux},
		{"rhel", OSTypeLinux},
		{"RHEL", OSTypeLinux},
		{"ubuntu", OSTypeLinux},
		{"CentOS", OSTypeLinux},
		{"solaris", OSTypeSolaris},
		{"Solaris", OSTypeSolaris},
		{"aix", OSTypeAIX},
		{"AIX", OSTypeAIX},
		{"unknownOS", OSType("unknownos")},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			var o OSType
			if err := json.Unmarshal([]byte(fmt.Sprintf("%q", tc.input)), &o); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if o != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, o)
			}
		})
	}
}
