"""LangChain integration example for SecureLens AI Gate."""

from securelens import SecureLensClient, AIGate
from securelens.integrations import SecureLensLangChainCallback, SecureLensRetriever

# Note: This example requires langchain to be installed
# pip install securelens[langchain] langchain langchain-openai


def callback_example():
    """Example using SecureLens as a LangChain callback."""
    print("=" * 60)
    print("LangChain Callback Example")
    print("=" * 60)

    try:
        from langchain.chains import RetrievalQA
        from langchain_openai import ChatOpenAI
        from langchain_community.vectorstores import Qdrant
        from langchain_openai import OpenAIEmbeddings
    except ImportError:
        print("LangChain not installed. Run: pip install securelens[langchain]")
        return

    # Initialize SecureLens
    client = SecureLensClient(
        api_key="sl_your_api_key_here",
        base_url="https://api.securelens.ai"
    )
    gate = AIGate(client)

    # Create the callback handler
    callback = SecureLensLangChainCallback(
        gate=gate,
        policies=["redact_pii", "audit_all"],
        block_on_violation=False,  # Set to True to raise exceptions
        redact_queries=True,
        redact_responses=True,
        metadata={
            "application": "customer_support",
            "environment": "production"
        }
    )

    # Setup LangChain components (example setup)
    llm = ChatOpenAI(
        model="gpt-4",
        temperature=0.7,
    )

    # In a real application, you would have your vector store configured
    # embeddings = OpenAIEmbeddings()
    # vectorstore = Qdrant.from_existing_collection(...)
    # retriever = vectorstore.as_retriever()

    # Create the chain with SecureLens callback
    # chain = RetrievalQA.from_chain_type(
    #     llm=llm,
    #     chain_type="stuff",
    #     retriever=retriever,
    #     callbacks=[callback]
    # )

    # Example query (would work with actual retriever)
    # result = chain.invoke({"query": "What is John Smith's account balance?"})

    # Access audit information after queries
    print(f"Audit IDs collected: {callback.audit_ids}")
    if callback.last_intercept_result:
        print(f"Last intercept result: {callback.last_intercept_result.audit_id}")

    client.close()


def retriever_wrapper_example():
    """Example using SecureLens retriever wrapper."""
    print("\n" + "=" * 60)
    print("SecureLens Retriever Wrapper Example")
    print("=" * 60)

    try:
        from langchain_community.vectorstores import FAISS
        from langchain_openai import OpenAIEmbeddings
        from langchain.schema import Document
    except ImportError:
        print("LangChain not installed. Run: pip install securelens[langchain]")
        return

    # Initialize SecureLens
    client = SecureLensClient(
        api_key="sl_your_api_key_here",
        base_url="https://api.securelens.ai"
    )
    gate = AIGate(client)

    # Create a sample vector store (in production, use your actual store)
    # embeddings = OpenAIEmbeddings()
    # documents = [
    #     Document(page_content="John Smith's salary is $150,000", metadata={"source": "hr"}),
    #     Document(page_content="Q4 revenue was $10M", metadata={"source": "finance"}),
    # ]
    # vectorstore = FAISS.from_documents(documents, embeddings)
    # base_retriever = vectorstore.as_retriever()

    # Wrap with SecureLens governance
    # secure_retriever = SecureLensRetriever(
    #     retriever=base_retriever,
    #     gate=gate,
    #     policies=["redact_pii", "redact_financial"],
    #     redact_content=True
    # )

    # Retrieved documents will have sensitive data redacted
    # docs = secure_retriever.get_relevant_documents("What is John's salary?")
    # for doc in docs:
    #     print(f"Content: {doc.page_content}")  # PII will be redacted

    client.close()


def full_chain_example():
    """Complete example with SecureLens integrated into a RAG chain."""
    print("\n" + "=" * 60)
    print("Full RAG Chain with SecureLens Example")
    print("=" * 60)

    try:
        from langchain.chains import RetrievalQA
        from langchain_openai import ChatOpenAI
        from langchain.prompts import PromptTemplate
    except ImportError:
        print("LangChain not installed. Run: pip install securelens[langchain]")
        return

    # Initialize SecureLens
    client = SecureLensClient(
        api_key="sl_your_api_key_here",
        base_url="https://api.securelens.ai"
    )
    gate = AIGate(client)

    # Custom prompt that includes governance context
    prompt_template = """You are a helpful assistant. Answer the question based on the context provided.
    
Important: This response is being monitored for data governance compliance.
Do not include any personally identifiable information in your response.

Context: {context}

Question: {question}

Answer:"""

    prompt = PromptTemplate(
        template=prompt_template,
        input_variables=["context", "question"]
    )

    # Create callback with strict policy enforcement
    callback = SecureLensLangChainCallback(
        gate=gate,
        policies=["redact_pii", "block_phi", "audit_all"],
        block_on_violation=True,  # Raise exception if policy violated
        metadata={
            "compliance_mode": "strict",
            "data_classification": "confidential"
        }
    )

    # Setup the chain (example - would need actual retriever)
    llm = ChatOpenAI(model="gpt-4", temperature=0)

    # chain = RetrievalQA.from_chain_type(
    #     llm=llm,
    #     chain_type="stuff",
    #     retriever=secure_retriever,  # Use wrapped retriever
    #     chain_type_kwargs={"prompt": prompt},
    #     callbacks=[callback]
    # )

    # Handle policy violations
    from securelens.exceptions import SecureLensPolicyError

    try:
        # result = chain.invoke({"query": "Show me patient medical records"})
        pass
    except SecureLensPolicyError as e:
        print(f"Query blocked by policy: {e.policy_name}")
        print(f"Violations: {e.violations}")
        # Log the violation, notify admin, etc.

    # Get all audit IDs for compliance reporting
    print(f"Session audit trail: {callback.audit_ids}")

    client.close()


def main():
    """Run all LangChain integration examples."""
    callback_example()
    retriever_wrapper_example()
    full_chain_example()

    print("\n" + "=" * 60)
    print("LangChain integration examples completed!")
    print("=" * 60)
    print("\nNote: These examples show the integration patterns.")
    print("Uncomment the actual LangChain code and configure your")
    print("vector store and API keys to run them.")


if __name__ == "__main__":
    main()
