package bench

import (
	"fmt"
	"strings"
)

// generateCurlFixture generates curl verbose output with headers and JSON response.
func generateCurlFixture() string {
	var lines []string
	lines = append(lines,
		"* Connected to api.example.com (93.184.216.34) port 443",
		"* TLS 1.3 connection using TLS_AES_256_GCM_SHA384",
		"* ALPN: server accepted h2",
		"* Server certificate:",
		"*   subject: CN=api.example.com",
		"*   issuer: C=US; O=Let's Encrypt; CN=R3",
		"*   SSL certificate verify ok.",
	)

	// Request headers
	lines = append(lines, "> POST /api/v2/users HTTP/2")
	for i := 0; i < 8; i++ {
		headers := []string{"Host: api.example.com", "User-Agent: curl/8.1.2",
			"Accept: application/json", "Content-Type: application/json",
			"Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.test", "X-Request-ID: req-abc123",
			"X-Correlation-ID: corr-def456", "Cache-Control: no-cache"}
		lines = append(lines, "> "+headers[i])
	}
	lines = append(lines, ">")

	// Response headers
	lines = append(lines, "< HTTP/2 200")
	responseHeaders := []string{
		"content-type: application/json; charset=utf-8",
		"date: Sat, 22 Mar 2026 10:00:00 GMT",
		"server: nginx/1.24.0", "x-request-id: req-abc123",
		"x-ratelimit-limit: 100", "x-ratelimit-remaining: 98",
		"cache-control: no-store", "vary: Accept-Encoding",
		"strict-transport-security: max-age=31536000",
		"x-content-type-options: nosniff",
	}
	for _, h := range responseHeaders {
		lines = append(lines, "< "+h)
	}
	lines = append(lines, "<")

	// Progress table
	lines = append(lines,
		"  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current",
		"                                 Dload  Upload   Total   Spent    Left  Speed",
		"100  2048  100  1536  100   512  12000   4000 --:--:-- --:--:-- --:--:-- 16000",
	)

	// Response body (JSON)
	lines = append(lines, `{"users":[`)
	for i := 0; i < 20; i++ {
		comma := ","
		if i == 19 {
			comma = ""
		}
		lines = append(lines, fmt.Sprintf(`  {"id":%d,"name":"User %d","email":"user%d@example.com","role":"admin","created":"2026-01-01T00:00:00Z"}%s`, i, i, i, comma))
	}
	lines = append(lines, `],"total":20,"page":1}`)

	return strings.Join(lines, "\n") + "\n"
}

// generatePythonFixture generates Python output with pip install + traceback.
func generatePythonFixture() string {
	var lines []string

	// pip satisfied lines
	packages := []string{"flask", "requests", "click", "itsdangerous", "jinja2",
		"werkzeug", "certifi", "charset-normalizer", "idna", "urllib3",
		"markupsafe", "blinker", "colorama", "packaging", "setuptools"}
	for _, pkg := range packages {
		lines = append(lines, fmt.Sprintf("Requirement already satisfied: %s in /usr/lib/python3/dist-packages (1.0.0)", pkg))
	}

	// Download/install
	lines = append(lines,
		"Collecting sqlalchemy==2.0.21",
		"Downloading SQLAlchemy-2.0.21.tar.gz (9.5 MB)",
		"Collecting psycopg2-binary==2.9.7",
		"Downloading psycopg2_binary-2.9.7.tar.gz (3.8 MB)",
		"Installing collected packages: psycopg2-binary, sqlalchemy",
		"Successfully installed psycopg2-binary-2.9.7 sqlalchemy-2.0.21",
		"",
	)

	// Deprecation warnings
	for i := 0; i < 5; i++ {
		lines = append(lines, fmt.Sprintf("/usr/lib/python3/module%d.py:%d: DeprecationWarning: old API usage", i, 10+i*5))
	}
	lines = append(lines, "")

	// Traceback (ImportError — should preserve full chain)
	lines = append(lines,
		"Traceback (most recent call last):",
		`  File "/app/main.py", line 10, in <module>`,
		"    from app.server import create_app",
		`  File "/app/server.py", line 5, in <module>`,
		"    from app.routes import register_routes",
		`  File "/app/routes.py", line 3, in <module>`,
		"    from app.handlers.auth import AuthHandler",
		`  File "/app/handlers/auth.py", line 2, in <module>`,
		"    from app.services.ldap import LDAPClient",
		`  File "/app/services/ldap.py", line 1, in <module>`,
		"    import ldap3",
		"ModuleNotFoundError: No module named 'ldap3'",
	)

	return strings.Join(lines, "\n") + "\n"
}

// generateTerraformFixture generates terraform plan output with refresh + changes.
func generateTerraformFixture() string {
	var lines []string

	// Refresh phase (40 resources)
	resources := []string{
		"aws_vpc.main", "aws_subnet.public[0]", "aws_subnet.public[1]",
		"aws_subnet.private[0]", "aws_subnet.private[1]",
		"aws_security_group.web", "aws_security_group.db", "aws_security_group.cache",
		"aws_instance.web[0]", "aws_instance.web[1]", "aws_instance.web[2]",
		"aws_instance.worker[0]", "aws_instance.worker[1]",
		"aws_rds_instance.main", "aws_rds_instance.replica",
		"aws_elasticache_cluster.redis", "aws_route53_record.web",
		"aws_route53_record.api", "aws_alb.web", "aws_alb_target_group.web",
		"aws_alb_listener.https", "aws_iam_role.lambda",
		"aws_iam_policy.lambda", "aws_lambda_function.processor",
		"aws_sqs_queue.tasks", "aws_sns_topic.alerts",
		"aws_cloudwatch_log_group.app", "aws_cloudwatch_alarm.cpu",
		"aws_s3_bucket.assets", "aws_s3_bucket.logs",
		"aws_cloudfront_distribution.cdn", "aws_acm_certificate.main",
		"aws_kms_key.data", "aws_secretsmanager_secret.db",
		"aws_ecs_cluster.main", "aws_ecs_service.api",
		"aws_ecs_task_definition.api", "aws_ecr_repository.api",
		"aws_nat_gateway.main", "aws_eip.nat",
	}
	for _, r := range resources {
		lines = append(lines, fmt.Sprintf("%s: Refreshing state... [id=%s-id-123]", r, r))
	}
	lines = append(lines, "")

	// Data sources
	for i := 0; i < 5; i++ {
		lines = append(lines, fmt.Sprintf("data.aws_ami.ubuntu[%d]: Reading...", i))
		lines = append(lines, fmt.Sprintf("data.aws_ami.ubuntu[%d]: Read complete after 1s [id=ami-abc%d]", i, i))
	}
	lines = append(lines, "")

	// Plan changes
	lines = append(lines,
		"Terraform used the selected providers to generate the following execution plan.",
		"Resource actions are indicated with the following symbols:",
		"  ~ update in-place",
		"  + create",
		"  - destroy",
		"",
		"  # aws_instance.web[0] will be updated in-place",
		`  ~ resource "aws_instance" "web" {`,
		`      ~ instance_type = "t3.micro" -> "t3.small"`,
		"        tags          = {",
		`            "Name" = "web-0"`,
		"        }",
		"    }",
		"",
		"  # aws_instance.web[1] will be updated in-place",
		`  ~ resource "aws_instance" "web" {`,
		`      ~ instance_type = "t3.micro" -> "t3.small"`,
		"    }",
		"",
		"  # aws_lambda_function.processor will be updated in-place",
		`  ~ resource "aws_lambda_function" "processor" {`,
		`      ~ runtime = "python3.9" -> "python3.12"`,
		`      ~ handler = "main.handler" -> "app.lambda_handler"`,
		"    }",
		"",
		"Plan: 0 to add, 3 to change, 0 to destroy.",
	)

	return strings.Join(lines, "\n") + "\n"
}

// generateKubectlFixture generates kubectl get/describe output with managedFields.
func generateKubectlFixture() string {
	var lines []string

	// Pod YAML with managedFields
	lines = append(lines,
		"apiVersion: v1",
		"kind: Pod",
		"metadata:",
		"  name: api-server-7f8b9c5d4e-x2k9m",
		"  namespace: production",
		"  managedFields:",
	)
	// Generate verbose managedFields (30 lines)
	for i := 0; i < 3; i++ {
		managers := []string{"kube-controller-manager", "kubelet", "kubectl-client-side-apply"}
		lines = append(lines,
			"    - apiVersion: v1",
			"      fieldsType: FieldsV1",
			"      fieldsV1:",
			"        f:metadata:",
			"          f:labels:",
			fmt.Sprintf("            f:app-%d: {}", i),
			"        f:spec:",
			"          f:containers:",
			fmt.Sprintf("      manager: %s", managers[i]),
			"      operation: Update",
			fmt.Sprintf("      time: \"2026-03-%02dT08:00:00Z\"", 20+i),
		)
	}
	// last-applied-configuration
	lines = append(lines,
		"  annotations:",
		"    kubectl.kubernetes.io/last-applied-configuration: |",
		`      {"apiVersion":"v1","kind":"Pod","metadata":{"name":"api-server","namespace":"production","labels":{"app":"api","version":"v2"}},"spec":{"containers":[{"name":"api","image":"myregistry.io/api:v2.3.1","ports":[{"containerPort":8080}],"resources":{"requests":{"cpu":"100m","memory":"128Mi"},"limits":{"cpu":"500m","memory":"512Mi"}},"env":[{"name":"DB_HOST","value":"db.internal"},{"name":"CACHE_HOST","value":"redis.internal"}],"livenessProbe":{"httpGet":{"path":"/health","port":8080},"initialDelaySeconds":10},"readinessProbe":{"httpGet":{"path":"/ready","port":8080}}}]}}`,
	)
	// Actual useful metadata
	lines = append(lines,
		"  labels:",
		"    app: api",
		"    version: v2",
		"    pod-template-hash: 7f8b9c5d4e",
		"spec:",
		"  containers:",
		"    - name: api",
		"      image: myregistry.io/api:v2.3.1",
		"      ports:",
		"        - containerPort: 8080",
		"          protocol: TCP",
		"      resources:",
		"        requests:",
		"          cpu: 100m",
		"          memory: 128Mi",
		"        limits:",
		"          cpu: 500m",
		"          memory: 512Mi",
		"      env:",
		"        - name: DB_HOST",
		"          value: db.internal",
		"        - name: CACHE_HOST",
		"          value: redis.internal",
		"  restartPolicy: Always",
		"status:",
		"  phase: Running",
		"  conditions:",
		"    - type: Ready",
		`      status: "True"`,
		"    - type: ContainersReady",
		`      status: "True"`,
	)

	return strings.Join(lines, "\n") + "\n"
}

// generateTarFixture generates tar verbose listing of 100 files.
func generateTarFixture() string {
	var lines []string
	dirs := []string{"project/src", "project/lib", "project/test", "project/docs",
		"project/scripts", "project/internal", "project/pkg", "project/cmd"}
	exts := []string{".go", ".py", ".js", ".ts", ".md", ".yaml", ".json", ".sh"}

	for i := 0; i < 100; i++ {
		dir := dirs[i%len(dirs)]
		ext := exts[i%len(exts)]
		size := 512 + (i*137)%8192
		lines = append(lines, fmt.Sprintf("-rw-r--r-- user/group %7d 2024-01-15 10:%02d %s/file_%03d%s",
			size, i%60, dir, i, ext))
	}
	return strings.Join(lines, "\n") + "\n"
}

// generateSSHFixture generates SSH output with banners, debug, and actual command output.
func generateSSHFixture() string {
	var lines []string

	// Debug lines
	for i := 0; i < 10; i++ {
		debugMsgs := []string{
			"Reading configuration data /etc/ssh/ssh_config",
			"Connecting to prod-server.example.com [10.0.1.50] port 22.",
			"Connection established.",
			"Remote protocol version 2.0, remote software version OpenSSH_8.9",
			"Authenticating to prod-server.example.com as 'deploy'",
			"Trying private key: /home/user/.ssh/id_ed25519",
			"Authentication succeeded (publickey).",
			"Requesting X11 forwarding with authentication spoofing.",
			"Entering interactive session.",
			"Sending environment.",
		}
		lines = append(lines, "debug1: "+debugMsgs[i])
	}

	// Banners
	lines = append(lines,
		"Warning: Permanently added 'prod-server.example.com' (ED25519) to the list of known hosts.",
		"###############################################",
		"#        Welcome to Production Server         #",
		"#   Unauthorized access is strictly prohibited#",
		"#   All actions are logged and monitored      #",
		"###############################################",
		"",
	)

	// Actual command output
	lines = append(lines,
		"CONTAINER ID   IMAGE                         STATUS          NAMES",
	)
	containers := []struct{ id, image, status, name string }{
		{"abc123def456", "myregistry.io/api:v2.3.1", "Up 2 days", "api-server"},
		{"bcd234ef5678", "myregistry.io/web:v1.8.0", "Up 2 days", "web-frontend"},
		{"cde345fa6789", "myregistry.io/worker:v1.5", "Up 2 days", "task-worker"},
		{"def456ab7890", "redis:7-alpine", "Up 5 days", "redis-cache"},
		{"ef5678bc9012", "postgres:15", "Up 5 days", "postgres-db"},
		{"fa6789cd0123", "nginx:1.25", "Up 5 days", "reverse-proxy"},
		{"0b789def1234", "prom/prometheus:v2.48", "Up 3 days", "prometheus"},
		{"1c890ef02345", "grafana/grafana:10.2", "Up 3 days", "grafana"},
	}
	for _, c := range containers {
		lines = append(lines, fmt.Sprintf("%-14s %-29s %-15s %s", c.id, c.image, c.status, c.name))
	}

	return strings.Join(lines, "\n") + "\n"
}

// generateNodeFixture generates Node.js runtime output with warnings and webpack noise.
func generateNodeFixture() string {
	var lines []string

	// Experimental warnings
	for i := 0; i < 5; i++ {
		warnings := []string{
			"ExperimentalWarning: The Fetch API is an experimental feature",
			"ExperimentalWarning: Custom ESM Loaders is an experimental feature",
			"ExperimentalWarning: Import assertions are not a stable feature",
			"ExperimentalWarning: buffer.Blob is an experimental feature",
			"ExperimentalWarning: The --experimental-loader is an experimental feature",
		}
		lines = append(lines, fmt.Sprintf("(node:%d) %s", 12345+i, warnings[i]))
	}

	// Deprecation warnings
	for i := 0; i < 8; i++ {
		lines = append(lines, fmt.Sprintf("(node:12345) [DEP%04d] DeprecationWarning: Deprecated API %d", i, i))
	}

	// Webpack build output
	lines = append(lines, "")
	for i := 0; i <= 100; i += 10 {
		lines = append(lines, fmt.Sprintf("  %d%% building modules (%d/500)", i, i*5))
	}
	lines = append(lines,
		"asset main.js 245 KiB [emitted] (name: main)",
		"asset vendor.js 1.2 MiB [emitted] (name: vendor)",
		"asset styles.css 45 KiB [emitted] (name: styles)",
		"asset main.js.map 890 KiB [emitted] (name: main)",
		"asset vendor.js.map 3.4 MiB [emitted] (name: vendor)",
		"modules by path ./src/ 120 KiB",
		"  modules by path ./src/components/ 45 KiB 23 modules",
		"  modules by path ./src/utils/ 12 KiB 8 modules",
		"  modules by path ./src/hooks/ 8 KiB 5 modules",
		"orphan modules 2.5 KiB [orphan] 3 modules",
		"runtime modules 5 KiB 10 modules",
		"cacheable modules 1.8 MiB",
		"",
		"webpack 5.89.0 compiled successfully in 4523 ms",
	)

	// App error with stack trace
	lines = append(lines,
		"",
		"TypeError: Cannot read properties of undefined (reading 'map')",
		"    at renderList (/app/src/components/UserList.tsx:15:23)",
		"    at processChild (/app/src/utils/renderer.ts:42:10)",
		"    at Module._compile (node:internal/modules/cjs/loader:1241:14)",
		"    at Module._extensions..js (node:internal/modules/cjs/loader:1295:10)",
		"    at Module.load (node:internal/modules/cjs/loader:1091:32)",
		"    at Module._load (node:internal/modules/cjs/loader:938:12)",
		"    at require (node:internal/modules/helpers:177:18)",
	)

	return strings.Join(lines, "\n") + "\n"
}
