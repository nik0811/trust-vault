"""Basic query example for SecureLens AI Gate."""

from securelens import SecureLensClient, AIGate

def main():
    # Initialize the client
    client = SecureLensClient(
        api_key="sl_your_api_key_here",
        base_url="https://api.securelens.ai"  # or your self-hosted URL
    )

    # Create the AI Gate
    gate = AIGate(client)

    # Example 1: Simple query interception
    print("=" * 50)
    print("Example 1: Query Interception")
    print("=" * 50)

    result = gate.intercept(
        query="What is John Smith's salary and social security number?",
        policies=["redact_pii"]
    )

    print(f"Original query: {result.original_query}")
    print(f"Safe query: {result.safe_query}")
    print(f"Blocked: {result.blocked}")
    print(f"Audit ID: {result.audit_id}")
    print(f"Processing time: {result.processing_time_ms}ms")

    if result.classifications:
        print("\nDetected classifications:")
        for c in result.classifications:
            print(f"  - {c.entity_type}: '{c.value}' (confidence: {c.confidence:.2f})")

    # Example 2: Classification only
    print("\n" + "=" * 50)
    print("Example 2: Classification Only")
    print("=" * 50)

    classifications = gate.classify(
        text="Contact me at john.doe@example.com or call 555-123-4567"
    )

    print("Detected entities:")
    for c in classifications:
        print(f"  - {c.entity_type}: '{c.value}' at position {c.start}-{c.end}")

    # Example 3: Query with policy violation
    print("\n" + "=" * 50)
    print("Example 3: Policy Enforcement")
    print("=" * 50)

    result = gate.intercept(
        query="Show me patient records for diagnosis codes",
        policies=["block_phi"]
    )

    if result.blocked:
        print(f"Query was blocked!")
        print(f"Reason: {result.block_reason}")
        for violation in result.policy_violations:
            print(f"  - Policy: {violation.policy_name}")
            print(f"    Action: {violation.action}")
            print(f"    Reason: {violation.reason}")
    else:
        print(f"Query allowed: {result.safe_query}")

    # Clean up
    client.close()


if __name__ == "__main__":
    main()
