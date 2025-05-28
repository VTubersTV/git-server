# Git Server

A Go-based server that provides GitHub repository statistics and contributor information for the VTubersTV organization. This server acts as a proxy and analytics service for GitHub repositories, offering cached statistics and contributor data.

## Features

- Repository statistics including stars, forks, contributors, and commits
- Top contributors listing with contribution counts
- Language-specific color coding
- Caching system for improved performance
- GitHub repository redirection
- RESTful API endpoints

## Prerequisites

- Go 1.24.3 or higher
- GitHub Personal Access Token with appropriate permissions

## Installation

1. Clone the repository:
```bash
git clone https://github.com/VTubersTV/git-server.git
cd git-server
```

2. Install dependencies:
```bash
go mod download
```

3. Create a `.env` file in the root directory with your GitHub token:
```env
GITHUB_TOKEN=your_github_token_here
```

## Usage

Start the server:
```bash
go run src/main.go
```

The server will start on port 8080 by default.

### API Endpoints

- `GET /`: Redirects to the VTubersTV GitHub organization
- `GET /:repo`: Redirects to the specific repository on GitHub
- `GET /stats`: Returns statistics for all public repositories
- `GET /contributors`: Returns contributor statistics
  - Query parameter: `limit` (optional) - Number of top contributors to return

### Example Response

Stats endpoint response:
```json
{
  "repositories": [
    {
      "name": "repo-name",
      "stars": 100,
      "forks": 50,
      "contributors": 10,
      "commits": 500,
      "license": "MIT",
      "last_updated": "2024-03-20T12:00:00Z",
      "description": "Repository description",
      "language": "Go",
      "language_color": "#00ADD8",
      "open_issues": 5,
      "default_branch": "main",
      "topics": ["go", "api", "server"]
    }
  ],
  "totalStars": 100,
  "totalForks": 50,
  "totalContributors": 10,
  "totalCommits": 500,
  "githubUrl": "https://github.com/VTubersTV/"
}
```

## Configuration

The server uses the following configuration:

- Cache duration: 15 minutes
- Default port: 8080
- GitHub base URL: https://github.com/VTubersTV/

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the AGPL-3.0 License and the VTubers.TV Commercial License (VCL) v1.0. See the [LICENSE](./LICENSE) and [LICENSE-VCL](./LICENSE-VCL.md) files for details.


