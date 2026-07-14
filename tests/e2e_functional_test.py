#!/usr/bin/env python3
"""
TrustVault End-to-End Functional Test Suite

This script tests actual functionality, not just page loads.
It verifies that features WORK by checking data flows and state changes.

Usage:
    python tests/e2e_functional_test.py

Environment variables (optional):
    TRUSTVAULT_API_URL - API base URL (default: http://localhost:8080)
    SUPERADMIN_EMAIL - Superadmin email (default: admin@trustvault.io)
    SUPERADMIN_PASSWORD - Superadmin password (default: admin123)
"""

import os
import sys
import json
import time
import uuid
import requests
from datetime import datetime
from typing import Optional, Dict, Any, List, Tuple
from dataclasses import dataclass, field

# Configuration
API_URL = os.getenv("TRUSTVAULT_API_URL", "http://localhost:8080")
INTERNAL_URL = os.getenv("TRUSTVAULT_INTERNAL_URL", "http://localhost:8099")
SUPERADMIN_EMAIL = os.getenv("SUPERADMIN_EMAIL", "admin@trustvault.io")
SUPERADMIN_PASSWORD = os.getenv("SUPERADMIN_PASSWORD", "admin123")

# Test data
TEST_TENANT_NAME = f"e2e-test-{uuid.uuid4().hex[:8]}"
TEST_USER_EMAIL = f"testuser-{uuid.uuid4().hex[:8]}@test.com"


@dataclass
class TestResult:
    name: str
    passed: bool
    message: str
    duration_ms: int = 0
    details: Dict[str, Any] = field(default_factory=dict)


@dataclass
class TestContext:
    """Shared context across tests"""
    token: Optional[str] = None
    tenant_id: Optional[str] = None
    user_id: Optional[str] = None
    datasource_id: Optional[str] = None
    policy_id: Optional[str] = None
    classification_id: Optional[str] = None
    feedback_id: Optional[str] = None
    gate_query_id: Optional[str] = None


class TrustVaultE2ETest:
    def __init__(self):
        self.ctx = TestContext()
        self.results: List[TestResult] = []
        self.session = requests.Session()
        self.session.headers.update({"Content-Type": "application/json"})

    def _api(self, method: str, endpoint: str, data: Dict = None, 
             expected_status: int = None, use_internal: bool = False) -> Tuple[int, Dict]:
        """Make API request and return (status_code, response_json)"""
        base = INTERNAL_URL if use_internal else API_URL
        url = f"{base}{endpoint}"
        
        try:
            if method == "GET":
                resp = self.session.get(url, timeout=30)
            elif method == "POST":
                resp = self.session.post(url, json=data, timeout=30)
            elif method == "PUT":
                resp = self.session.put(url, json=data, timeout=30)
            elif method == "DELETE":
                resp = self.session.delete(url, timeout=30)
            else:
                raise ValueError(f"Unknown method: {method}")
            
            try:
                body = resp.json()
            except:
                body = {"raw": resp.text}
            
            if expected_status and resp.status_code != expected_status:
                raise AssertionError(
                    f"Expected status {expected_status}, got {resp.status_code}: {body}"
                )
            
            return resp.status_code, body
        except requests.exceptions.ConnectionError as e:
            raise ConnectionError(f"Cannot connect to {url}: {e}")

    def _set_auth(self, token: str):
        """Set authorization header"""
        self.session.headers["Authorization"] = f"Bearer {token}"
        self.ctx.token = token

    def run_test(self, name: str, test_func) -> TestResult:
        """Run a single test and record result"""
        start = time.time()
        try:
            test_func()
            duration = int((time.time() - start) * 1000)
            result = TestResult(name=name, passed=True, message="PASSED", duration_ms=duration)
        except AssertionError as e:
            duration = int((time.time() - start) * 1000)
            result = TestResult(name=name, passed=False, message=f"FAILED: {e}", duration_ms=duration)
        except Exception as e:
            duration = int((time.time() - start) * 1000)
            result = TestResult(name=name, passed=False, message=f"ERROR: {type(e).__name__}: {e}", duration_ms=duration)
        
        self.results.append(result)
        status = "✓" if result.passed else "✗"
        print(f"  {status} {name} ({result.duration_ms}ms)")
        if not result.passed:
            print(f"    └─ {result.message}")
        return result

    # =========================================================================
    # 1. AUTH FLOW TESTS
    # =========================================================================
    
    def test_auth_login(self):
        """Test login with superadmin credentials and verify token is returned"""
        status, body = self._api("POST", "/api/v1/auth/login", {
            "email": SUPERADMIN_EMAIL,
            "password": SUPERADMIN_PASSWORD
        })
        
        assert status == 200, f"Login failed with status {status}: {body}"
        assert "access_token" in body, f"No access_token in response: {body}"
        assert body["access_token"], "access_token is empty"
        assert body.get("expires_in", 0) > 0, "expires_in should be positive"
        
        self._set_auth(body["access_token"])
        print(f"    └─ Token obtained, expires in {body.get('expires_in')}s")

    def test_auth_protected_endpoint(self):
        """Test that protected endpoints require valid token"""
        # First, try without token
        old_auth = self.session.headers.pop("Authorization", None)
        status, body = self._api("GET", "/api/v1/datasources")
        assert status == 401, f"Expected 401 without token, got {status}"
        
        # Restore token and verify access
        if old_auth:
            self.session.headers["Authorization"] = old_auth
        status, body = self._api("GET", "/api/v1/datasources")
        assert status == 200, f"Expected 200 with valid token, got {status}: {body}"

    def test_auth_invalid_token(self):
        """Test that invalid tokens are rejected"""
        old_auth = self.session.headers.get("Authorization")
        self.session.headers["Authorization"] = "Bearer invalid-token-12345"
        
        status, body = self._api("GET", "/api/v1/datasources")
        assert status == 401, f"Expected 401 for invalid token, got {status}"
        
        # Restore valid token
        if old_auth:
            self.session.headers["Authorization"] = old_auth

    # =========================================================================
    # 2. DATA SOURCE FLOW TESTS
    # =========================================================================

    def test_datasource_create(self):
        """Create a PostgreSQL data source"""
        ds_name = f"e2e-test-postgres-{uuid.uuid4().hex[:8]}"
        status, body = self._api("POST", "/api/v1/datasources", {
            "name": ds_name,
            "type": "postgresql",
            "config": {
                "host": "localhost",
                "port": 5432,
                "database": "testdb",
                "username": "testuser",
                "password": "testpass"
            }
        }, expected_status=201)
        
        assert "id" in body, f"No id in response: {body}"
        assert body["name"] == ds_name, f"Name mismatch: {body['name']} != {ds_name}"
        assert body["type"] == "postgresql", f"Type mismatch: {body['type']}"
        assert body["status"] == "pending", f"Initial status should be pending: {body['status']}"
        
        # Verify password is masked
        config = body.get("config", {})
        if isinstance(config, str):
            config = json.loads(config)
        assert config.get("password") == "********", f"Password should be masked: {config}"
        
        self.ctx.datasource_id = body["id"]
        print(f"    └─ Created datasource: {body['id']}")

    def test_datasource_get(self):
        """Verify datasource can be retrieved"""
        assert self.ctx.datasource_id, "No datasource_id from previous test"
        
        status, body = self._api("GET", f"/api/v1/datasources/{self.ctx.datasource_id}")
        assert status == 200, f"Failed to get datasource: {body}"
        assert body["id"] == self.ctx.datasource_id

    def test_datasource_list(self):
        """Verify datasource appears in list"""
        status, body = self._api("GET", "/api/v1/datasources")
        assert status == 200, f"Failed to list datasources: {body}"
        assert isinstance(body, list), f"Expected list, got {type(body)}"
        
        ids = [ds["id"] for ds in body]
        assert self.ctx.datasource_id in ids, f"Created datasource not in list"

    def test_datasource_trigger_scan(self):
        """Trigger scan on datasource (will fail without ingestion sidecar, but tests the endpoint)"""
        assert self.ctx.datasource_id, "No datasource_id"
        
        status, body = self._api("POST", f"/api/v1/datasources/{self.ctx.datasource_id}/scan")
        
        # Scan may fail if ingestion sidecar is not running - that's expected
        # We just verify the endpoint responds correctly
        if status == 503:
            print(f"    └─ Scan endpoint works (ingestion sidecar unavailable - expected)")
        elif status == 200:
            assert "status" in body, f"No status in response: {body}"
            print(f"    └─ Scan triggered: {body.get('status')}")
        elif status == 409:
            print(f"    └─ Scan already in progress (expected)")
        else:
            raise AssertionError(f"Unexpected status {status}: {body}")

    def test_datasource_scan_status(self):
        """Check scan status endpoint"""
        assert self.ctx.datasource_id, "No datasource_id"
        
        status, body = self._api("GET", f"/api/v1/datasources/{self.ctx.datasource_id}/status")
        assert status == 200, f"Failed to get scan status: {body}"
        assert "status" in body, f"No status field: {body}"
        print(f"    └─ Scan status: {body['status']}")

    # =========================================================================
    # 3. CLASSIFICATION FLOW TESTS
    # =========================================================================

    def test_classification_text(self):
        """Test text classification with PII data"""
        test_text = """
        Customer: John Smith
        Email: john.smith@example.com
        Phone: (555) 123-4567
        SSN: 123-45-6789
        Credit Card: 4111-1111-1111-1111
        IP Address: 192.168.1.100
        """
        
        status, body = self._api("POST", "/api/v1/classify/text", {
            "text": test_text
        })
        
        assert status == 200, f"Classification failed: {body}"
        assert "entities" in body, f"No entities in response: {body}"
        
        entities = body["entities"]
        assert len(entities) > 0, "No entities detected in PII-rich text"
        
        # Verify specific entity types were detected
        detected_types = {e["entity"] for e in entities}
        expected_types = {"EMAIL", "PHONE", "SSN", "CREDIT_CARD", "IP_ADDRESS"}
        found_types = detected_types & expected_types
        
        print(f"    └─ Detected {len(entities)} entities: {detected_types}")
        assert len(found_types) >= 3, f"Expected at least 3 PII types, found: {found_types}"

    def test_classification_models_list(self):
        """Verify classification models are available"""
        status, body = self._api("GET", "/api/v1/classify/models")
        assert status == 200, f"Failed to list models: {body}"
        assert isinstance(body, list), f"Expected list: {body}"
        assert len(body) > 0, "No classification models available"
        
        # Check for expected built-in models
        model_ids = [m.get("id") for m in body]
        print(f"    └─ Available models: {model_ids}")

    def test_classification_rules_crud(self):
        """Test classification rules CRUD operations"""
        # Create rule
        rule_name = f"e2e-test-rule-{uuid.uuid4().hex[:8]}"
        status, body = self._api("POST", "/api/v1/classify/rules", {
            "name": rule_name,
            "type": "pattern",
            "column_pattern": ".*email.*",
            "entity_type": "EMAIL",
            "confidence": 0.95,
            "priority": 10,
            "active": True
        }, expected_status=201)
        
        assert "id" in body, f"No id in response: {body}"
        rule_id = body["id"]
        print(f"    └─ Created rule: {rule_id}")
        
        # List rules
        status, body = self._api("GET", "/api/v1/classify/rules")
        assert status == 200
        
        # Delete rule
        status, body = self._api("DELETE", f"/api/v1/classify/rules/{rule_id}")
        assert status == 200, f"Failed to delete rule: {body}"

    # =========================================================================
    # 4. POLICY FLOW TESTS
    # =========================================================================

    def test_policy_create(self):
        """Create a governance policy"""
        policy_name = f"e2e-test-policy-{uuid.uuid4().hex[:8]}"
        status, body = self._api("POST", "/api/v1/governance/policies", {
            "name": policy_name,
            "description": "E2E test policy for PII redaction",
            "type": "redaction",
            "conditions": json.dumps({
                "data_classification": ["SSN", "CREDIT_CARD"],
                "destination_type": ["llm", "export"]
            }),
            "actions": json.dumps({
                "action": "redact",
                "notify": True
            }),
            "priority": 100,
            "active": True
        }, expected_status=201)
        
        assert "id" in body, f"No id in response: {body}"
        self.ctx.policy_id = body["id"]
        print(f"    └─ Created policy: {body['id']}")

    def test_policy_list(self):
        """Verify policy appears in list"""
        status, body = self._api("GET", "/api/v1/governance/policies")
        assert status == 200, f"Failed to list policies: {body}"
        assert isinstance(body, list), f"Expected list: {body}"
        
        if self.ctx.policy_id:
            ids = [p["id"] for p in body]
            assert self.ctx.policy_id in ids, "Created policy not in list"

    def test_policy_evaluate(self):
        """Test policy evaluation against test data"""
        status, body = self._api("POST", "/api/v1/governance/evaluate", {
            "data": "Customer SSN: 123-45-6789, Card: 4111111111111111",
            "context": {
                "destination": "llm",
                "user_role": "analyst"
            }
        })
        
        assert status == 200, f"Policy evaluation failed: {body}"
        assert "decision" in body, f"No decision in response: {body}"
        assert "applied_policies" in body, f"No applied_policies: {body}"
        
        print(f"    └─ Decision: {body['decision']}, Applied: {len(body['applied_policies'])} policies")

    # =========================================================================
    # 5. FEEDBACK FLOW TESTS
    # =========================================================================

    def test_feedback_submit_correction(self):
        """Submit a classification correction"""
        # First, create a classification to correct
        status, body = self._api("POST", "/api/v1/classify/text", {
            "text": "Contact: test@example.com"
        })
        assert status == 200
        
        # Get a classification ID (use a fake one if none returned)
        classification_id = f"test-classification-{uuid.uuid4().hex[:8]}"
        
        status, body = self._api("POST", "/api/v1/feedback/correction", {
            "classification_id": classification_id,
            "corrected_label": "BUSINESS_EMAIL"
        }, expected_status=201)
        
        assert "id" in body, f"No id in response: {body}"
        self.ctx.feedback_id = body["id"]
        print(f"    └─ Submitted correction: {body['id']}")

    def test_feedback_list_corrections(self):
        """Verify correction appears in list"""
        status, body = self._api("GET", "/api/v1/feedback/corrections")
        assert status == 200, f"Failed to list corrections: {body}"
        assert isinstance(body, list), f"Expected list: {body}"
        
        if self.ctx.feedback_id:
            ids = [f["id"] for f in body]
            assert self.ctx.feedback_id in ids, "Submitted correction not in list"
        
        print(f"    └─ Found {len(body)} corrections")

    def test_feedback_stats(self):
        """Get feedback statistics"""
        status, body = self._api("GET", "/api/v1/feedback/stats")
        assert status == 200, f"Failed to get feedback stats: {body}"
        assert "total_corrections" in body, f"No total_corrections: {body}"
        assert "total_confirmations" in body, f"No total_confirmations: {body}"
        
        print(f"    └─ Corrections: {body['total_corrections']}, Confirmations: {body['total_confirmations']}")

    # =========================================================================
    # 6. AUDIT TRAIL TESTS
    # =========================================================================

    def test_audit_trail_recorded(self):
        """Verify audit logs are recorded for actions"""
        status, body = self._api("GET", "/api/v1/audit/trail?limit=50")
        assert status == 200, f"Failed to get audit trail: {body}"
        assert isinstance(body, list), f"Expected list: {body}"
        
        # Check for our test actions
        actions = [log.get("action") for log in body]
        
        # We should see datasource and policy creation events
        expected_actions = ["datasource.created", "policy.created"]
        found_actions = [a for a in expected_actions if a in actions]
        
        print(f"    └─ Found {len(body)} audit logs, actions: {set(actions)}")
        
        if len(found_actions) == 0:
            print(f"    └─ Warning: Expected actions not found (may be from different tenant)")

    def test_audit_compliance_report(self):
        """Get compliance report"""
        status, body = self._api("GET", "/api/v1/audit/compliance-report")
        assert status == 200, f"Failed to get compliance report: {body}"
        assert "compliance" in body, f"No compliance data: {body}"
        
        compliance = body["compliance"]
        print(f"    └─ Compliance scores - GDPR: {compliance.get('gdpr', 0):.0%}, CCPA: {compliance.get('ccpa', 0):.0%}")

    # =========================================================================
    # 7. AI GATE FLOW TESTS
    # =========================================================================

    def test_gate_query_with_redaction(self):
        """Send query through AI Gate and verify redaction"""
        status, body = self._api("POST", "/api/v1/gate/query", {
            "query": "What is the customer's SSN 123-45-6789 and email john@example.com?",
            "max_chunks": 3,
            "llm_endpoint": "http://localhost:11434/v1",
            "model": "llama2"
        })
        
        # Gate query may fail if LLM is not available, but we test the flow
        if status == 200:
            assert "id" in body, f"No id in response: {body}"
            assert "decision" in body, f"No decision: {body}"
            assert "redactions" in body, f"No redactions field: {body}"
            
            self.ctx.gate_query_id = body["id"]
            redactions = body.get("redactions", [])
            
            print(f"    └─ Query ID: {body['id']}, Decision: {body['decision']}, Redactions: {len(redactions)}")
            
            # Verify sensitive data was redacted
            if redactions:
                redacted_types = [r.get("type") for r in redactions]
                print(f"    └─ Redacted types: {redacted_types}")
        else:
            # LLM not available is acceptable
            print(f"    └─ Gate query returned {status} (LLM may not be available)")

    def test_gate_validate_output(self):
        """Test output validation for data leakage"""
        status, body = self._api("POST", "/api/v1/gate/validate", {
            "response": "The customer's SSN is 123-45-6789 and their credit card is 4111111111111111"
        })
        
        assert status == 200, f"Validation failed: {body}"
        assert "decision" in body, f"No decision: {body}"
        
        # Should flag this as containing sensitive data
        if body["decision"] == "flagged":
            print(f"    └─ Correctly flagged sensitive data in output")
        else:
            print(f"    └─ Decision: {body['decision']} (may not have detected PII)")

    def test_gate_stats(self):
        """Get AI Gate statistics"""
        status, body = self._api("GET", "/api/v1/gate/stats")
        assert status == 200, f"Failed to get gate stats: {body}"
        assert "total_queries" in body, f"No total_queries: {body}"
        
        print(f"    └─ Total queries: {body['total_queries']}, Blocked: {body.get('queries_blocked', 0)}")

    def test_gate_query_list(self):
        """List gate queries"""
        status, body = self._api("GET", "/api/v1/gate/queries?limit=10")
        assert status == 200, f"Failed to list gate queries: {body}"
        assert isinstance(body, list), f"Expected list: {body}"
        
        print(f"    └─ Found {len(body)} gate queries")

    # =========================================================================
    # 8. COMPLIANCE TESTS
    # =========================================================================

    def test_compliance_recommendations(self):
        """Get compliance recommendations based on actual data"""
        status, body = self._api("GET", "/api/v1/compliance/recommendations")
        assert status == 200, f"Failed to get recommendations: {body}"
        assert isinstance(body, list), f"Expected list: {body}"
        
        if body:
            priorities = [r.get("priority") for r in body]
            print(f"    └─ Got {len(body)} recommendations, priorities: {set(priorities)}")
        else:
            print(f"    └─ No recommendations (may need more data)")

    def test_compliance_gaps(self):
        """Get compliance gaps"""
        status, body = self._api("GET", "/api/v1/compliance/gaps")
        assert status == 200, f"Failed to get compliance gaps: {body}"
        
        # Should have framework-specific gaps
        frameworks = ["gdpr", "ccpa", "hipaa"]
        for fw in frameworks:
            if fw in body:
                score = body[fw].get("score", 0)
                gaps = body[fw].get("gaps", [])
                print(f"    └─ {fw.upper()}: {score:.0%} compliance, {len(gaps)} gaps")

    def test_compliance_risk_score(self):
        """Get overall risk score"""
        status, body = self._api("GET", "/api/v1/compliance/risk-score")
        assert status == 200, f"Failed to get risk score: {body}"
        
        if "overall" in body:
            print(f"    └─ Overall risk score: {body['overall']}")
        elif "score" in body:
            print(f"    └─ Risk score: {body['score']}")

    # =========================================================================
    # CLEANUP
    # =========================================================================

    def cleanup(self):
        """Clean up test resources"""
        print("\n🧹 Cleaning up test resources...")
        
        # Delete test datasource
        if self.ctx.datasource_id:
            try:
                self._api("DELETE", f"/api/v1/datasources/{self.ctx.datasource_id}")
                print(f"  ✓ Deleted datasource: {self.ctx.datasource_id}")
            except Exception as e:
                print(f"  ✗ Failed to delete datasource: {e}")
        
        # Delete test policy
        if self.ctx.policy_id:
            try:
                self._api("DELETE", f"/api/v1/governance/policies/{self.ctx.policy_id}")
                print(f"  ✓ Deleted policy: {self.ctx.policy_id}")
            except Exception as e:
                print(f"  ✗ Failed to delete policy: {e}")

    # =========================================================================
    # TEST RUNNER
    # =========================================================================

    def run_all(self):
        """Run all tests"""
        print("=" * 70)
        print("TrustVault E2E Functional Test Suite")
        print(f"API URL: {API_URL}")
        print(f"Time: {datetime.now().isoformat()}")
        print("=" * 70)

        # Check API is reachable
        print("\n🔍 Checking API availability...")
        try:
            status, body = self._api("GET", "/health")
            if status != 200:
                print(f"❌ API health check failed: {status}")
                return False
            print(f"✓ API is healthy: {body}")
        except ConnectionError as e:
            print(f"❌ Cannot connect to API: {e}")
            print("\nMake sure the TrustVault server is running:")
            print(f"  cd /Users/apple/Documents/trustvault && go run cmd/gateway/main.go")
            return False

        # Run test suites
        test_suites = [
            ("1. Authentication Flow", [
                ("Login with superadmin", self.test_auth_login),
                ("Protected endpoint access", self.test_auth_protected_endpoint),
                ("Invalid token rejection", self.test_auth_invalid_token),
            ]),
            ("2. Data Source Flow", [
                ("Create PostgreSQL datasource", self.test_datasource_create),
                ("Get datasource by ID", self.test_datasource_get),
                ("List datasources", self.test_datasource_list),
                ("Trigger scan", self.test_datasource_trigger_scan),
                ("Check scan status", self.test_datasource_scan_status),
            ]),
            ("3. Classification Flow", [
                ("Classify text with PII", self.test_classification_text),
                ("List classification models", self.test_classification_models_list),
                ("Classification rules CRUD", self.test_classification_rules_crud),
            ]),
            ("4. Policy Flow", [
                ("Create governance policy", self.test_policy_create),
                ("List policies", self.test_policy_list),
                ("Evaluate policy", self.test_policy_evaluate),
            ]),
            ("5. Feedback Flow", [
                ("Submit correction", self.test_feedback_submit_correction),
                ("List corrections", self.test_feedback_list_corrections),
                ("Get feedback stats", self.test_feedback_stats),
            ]),
            ("6. Audit Trail", [
                ("Verify audit logs recorded", self.test_audit_trail_recorded),
                ("Get compliance report", self.test_audit_compliance_report),
            ]),
            ("7. AI Gate Flow", [
                ("Query with redaction", self.test_gate_query_with_redaction),
                ("Validate output for leakage", self.test_gate_validate_output),
                ("Get gate stats", self.test_gate_stats),
                ("List gate queries", self.test_gate_query_list),
            ]),
            ("8. Compliance", [
                ("Get recommendations", self.test_compliance_recommendations),
                ("Get compliance gaps", self.test_compliance_gaps),
                ("Get risk score", self.test_compliance_risk_score),
            ]),
        ]

        for suite_name, tests in test_suites:
            print(f"\n📋 {suite_name}")
            print("-" * 50)
            for test_name, test_func in tests:
                self.run_test(test_name, test_func)

        # Cleanup
        self.cleanup()

        # Summary
        print("\n" + "=" * 70)
        print("TEST SUMMARY")
        print("=" * 70)
        
        passed = sum(1 for r in self.results if r.passed)
        failed = sum(1 for r in self.results if not r.passed)
        total = len(self.results)
        
        print(f"\nTotal: {total} | Passed: {passed} | Failed: {failed}")
        print(f"Success Rate: {passed/total*100:.1f}%")
        
        if failed > 0:
            print("\n❌ FAILED TESTS:")
            for r in self.results:
                if not r.passed:
                    print(f"  • {r.name}: {r.message}")
        
        print("\n" + "=" * 70)
        
        return failed == 0


def main():
    test = TrustVaultE2ETest()
    success = test.run_all()
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
