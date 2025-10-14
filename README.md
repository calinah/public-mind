# 🏛️ PublicMind CVRD

> *"Taking the guesswork out of local developments."*

PublicMind is a civic transparency AI assistant that helps residents understand **developments, plans, and policies that impact their community**. It makes local governance transparent and accessible — especially for seniors and non-technical users.

## 🌟 What is PublicMind?

PublicMind transforms complex civic documents (OCPs, bylaws, council minutes, etc.) into an easy-to-use Q&A interface. Simply ask natural-language questions about local developments and get grounded, sourced answers from official documents.

**Example questions:**
- "What are the plans for the Youbou industrial site?"
- "What are the new zoning regulations for residential areas?"
- "When is the next public hearing on the waterfront development?"

## 🚀 Quick Start

### For Users
1. Visit the PublicMind website
2. Enter your question in the search box
3. Get an AI-powered answer with official sources
4. Use the "Read Aloud" feature for accessibility

### For Developers

#### Prerequisites
- Go 1.21+
- OpenAI API key
- Vector database (Supabase pgvector or Qdrant)

#### Setup
1. **Clone the repository**
   ```bash
   git clone https://github.com/calinah/public-mind.git
   cd public-mind
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Configure environment**
   ```bash
   cp .env.example .env
   # Edit .env with your API keys and database URLs
   ```

4. **Download civic documents**
   
   The civic documents are stored separately due to file size.
   
   **[📥 Download Documents from Google Drive](https://drive.google.com/drive/folders/1eCagIklBbQMWiChX8QTuywmgyNBNw7nX?usp=sharing)**
   
   Extract the files to the `docs/` folder in your project:
   ```
   public-mind/
   ├── docs/           ← Put PDFs here
   │   ├── Bylaw No. 4373 Schedule A.pdf
   │   ├── PostMinutes - Package.pdf
   │   └── ...
   ```

5. **Ingest documents** (first time setup)
   ```bash
   go run cmd/ingest/main.go
   ```

6. **Run the development server**
   ```bash
   go run cmd/server/main.go
   ```

## 🏗️ Architecture

```
public-mind/
├── cmd/
│   ├── server/     # Go HTTP API handling /ask
│   └── ingest/     # Go CLI for document ingestion + embeddings
├── internal/
│   ├── retriever/  # Vector DB client
│   ├── llm/        # OpenAI API wrapper
│   ├── chunker/    # Chunking + token utilities
│   ├── config/     # Environment variables
│   ├── middleware/ # Logging, rate limiting, validation
│   └── models/     # Data structures
├── web/            # Static frontend
├── docs/           # Uploaded civic documents (PDFs)
└── evals/          # QA test dataset
```

## 🛠️ Development

### Backend (Go)
- **Framework:** Gin or Fiber
- **Endpoints:**
  - `POST /ask` — question → retrieval → LLM → answer
  - `GET /healthz` — health check
- **Features:** Request logging, IP rate limiting, error handling, in-memory caching

### Frontend
- Simple static site (HTML + JS / React)
- High-contrast, large fonts for accessibility
- Single textbox + "Ask" button
- Answer display with sources and read-aloud functionality

### Document Processing
- **Chunking:** 900 tokens with 150 token overlap
- **Embeddings:** OpenAI `text-embedding-3-small`
- **Retrieval:** Top-5 most relevant chunks
- **LLM:** GPT-4o-mini for cost efficiency

## 📊 Configuration

| Setting | Default Value |
|---------|---------------|
| Embedding model | `text-embedding-3-small` |
| Chunk size | 900 tokens (+150 overlap) |
| Top-k retrieval | 5 chunks |
| Context limit | 4000 tokens |
| Completion model | `gpt-4o-mini` |
| Cache duration | 24 hours per query |

## 🚀 Deployment

### Backend
- **Platform:** Fly.io or Render (free tier)
- **Database:** Supabase pgvector
- **Monitoring:** Console logs + optional Grafana Cloud

### Frontend
- **Platform:** Cloudflare Pages (free static hosting)
- **CDN:** Global edge deployment

### CI/CD
- **GitHub Actions:** Automated deployment and document ingestion
- **Triggers:** Push to main branch

## 🔐 Privacy & Security

- **No user accounts** or persistent personal data
- **Anonymized logging** (hashed IP + timestamp)
- **Public documents only** — no private or sensitive information
- **Rate limiting** to prevent abuse
- **Open source** — fully auditable code

## 🧪 Testing

Run the evaluation suite:
```bash
go run cmd/eval/main.go
```

This tests:
- **Recall@k** — How often relevant chunks are retrieved
- **Faithfulness** — Accuracy of AI responses vs. source documents

## 📁 Adding Documents

1. **Add PDFs** to the `docs/` directory
2. **Run ingestion:**
   ```bash
   go run cmd/ingest/main.go
   ```
3. **Verify** documents are indexed and searchable

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👥 Authors

- **Francis Hall** — Original idea & custom GPT
- **Ana Calin** — System design & implementation

## 🎯 Mission

**Community transparency • Open-source MLOps practice • Civic accessibility**

PublicMind is designed to make local government more accessible to everyone, especially seniors and non-technical users who may struggle with complex civic documents and processes.

---

*Built with ❤️ for community transparency*
