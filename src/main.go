package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"

	"git.vtubers.tv/src/colors"
	"github.com/joho/godotenv"
)

const (
	githubBaseURL = "https://github.com/VTubersTV/"
	cacheDuration = 15 * time.Minute
)

type RepoStats struct {
	Name          string    `json:"name"`
	Stars         int       `json:"stars"`
	Forks         int       `json:"forks"`
	Contributors  int       `json:"contributors"`
	Commits       int       `json:"commits"`
	License       string    `json:"license"`
	LastUpdated   time.Time `json:"last_updated"`
	Description   string    `json:"description"`
	Language      string    `json:"language"`
	LanguageColor string    `json:"language_color"`
	OpenIssues    int       `json:"open_issues"`
	DefaultBranch string    `json:"default_branch"`
	Tags          []string  `json:"topics"`
}

type ContributorStats struct {
	Login         string   `json:"login"`
	AvatarURL     string   `json:"avatar_url"`
	Contributions int      `json:"contributions"`
	Repositories  []string `json:"repositories"`
}

type CacheItem struct {
	Data      interface{}
	Timestamp time.Time
}

type Cache struct {
	stats        CacheItem
	contributors CacheItem
	mu           sync.RWMutex
}

var cache = &Cache{}

func (c *Cache) isStale(item *CacheItem) bool {
	return time.Since(item.Timestamp) > cacheDuration
}

func (c *Cache) updateStats(stats []RepoStats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stats = CacheItem{
		Data:      stats,
		Timestamp: time.Now(),
	}
}

func (c *Cache) getStats() ([]RepoStats, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.stats.Data == nil {
		return nil, false
	}
	return c.stats.Data.([]RepoStats), true
}

func (c *Cache) updateContributors(contributors []ContributorStats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.contributors = CacheItem{
		Data:      contributors,
		Timestamp: time.Now(),
	}
}

func (c *Cache) getContributors() ([]ContributorStats, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.contributors.Data == nil {
		return nil, false
	}
	return c.contributors.Data.([]ContributorStats), true
}

func fetchStats(client *github.Client) ([]RepoStats, error) {
	ctx := context.Background()
	repos, _, err := client.Repositories.ListByOrg(ctx, "VTubersTV", &github.RepositoryListByOrgOptions{
		Type: "all",
	})
	if err != nil {
		return nil, err
	}

	var allStats []RepoStats
	for _, repo := range repos {
		// Skip private repositories
		if repo.GetPrivate() {
			continue
		}

		// Get contributors count
		contributors, _, err := client.Repositories.ListContributors(ctx, "VTubersTV", *repo.Name, &github.ListContributorsOptions{})
		if err != nil {
			log.Printf("Error getting contributors for %s: %v", *repo.Name, err)
		}

		// Get commit count
		commits, _, err := client.Repositories.ListCommits(ctx, "VTubersTV", *repo.Name, &github.CommitsListOptions{})
		if err != nil {
			log.Printf("Error getting commits for %s: %v", *repo.Name, err)
		}

		// Get tags
		tags, _, err := client.Repositories.ListAllTopics(ctx, "VTubersTV", *repo.Name)
		if err != nil {
			log.Printf("Error getting tags for %s: %v", *repo.Name, err)
		}

		// Convert tags to strings
		tagNames := make([]string, len(tags))
		copy(tagNames, tags)

		language := repo.GetLanguage()
		stats := RepoStats{
			Name:          *repo.Name,
			Stars:         *repo.StargazersCount,
			Forks:         *repo.ForksCount,
			Contributors:  len(contributors),
			Commits:       len(commits),
			License:       repo.GetLicense().GetName(),
			LastUpdated:   repo.GetUpdatedAt().Time,
			Description:   repo.GetDescription(),
			Language:      language,
			LanguageColor: colors.GetLanguageColor(language),
			OpenIssues:    repo.GetOpenIssuesCount(),
			DefaultBranch: repo.GetDefaultBranch(),
			Tags:          tagNames,
		}
		allStats = append(allStats, stats)
	}

	return allStats, nil
}

func fetchTopContributors(client *github.Client, limit int) ([]ContributorStats, error) {
	ctx := context.Background()
	repos, _, err := client.Repositories.ListByOrg(ctx, "VTubersTV", &github.RepositoryListByOrgOptions{
		Type: "all",
	})
	if err != nil {
		return nil, err
	}

	// Map to store unique contributors and their total contributions
	contributorMap := make(map[string]*ContributorStats)

	for _, repo := range repos {
		// Skip private repositories
		if repo.GetPrivate() {
			continue
		}

		contributors, _, err := client.Repositories.ListContributors(ctx, "VTubersTV", *repo.Name, &github.ListContributorsOptions{})
		if err != nil {
			log.Printf("Error getting contributors for %s: %v", *repo.Name, err)
			continue
		}

		for _, contributor := range contributors {
			login := contributor.GetLogin()
			if login == "" || strings.Contains(strings.ToLower(login), "[bot]") {
				continue
			}

			if stats, exists := contributorMap[login]; exists {
				stats.Contributions += contributor.GetContributions()
				stats.Repositories = append(stats.Repositories, *repo.Name)
			} else {
				contributorMap[login] = &ContributorStats{
					Login:         login,
					AvatarURL:     contributor.GetAvatarURL(),
					Contributions: contributor.GetContributions(),
					Repositories:  []string{*repo.Name},
				}
			}
		}
	}

	// Convert map to slice and sort by contributions
	var contributors []ContributorStats
	for _, stats := range contributorMap {
		contributors = append(contributors, *stats)
	}

	sort.Slice(contributors, func(i, j int) bool {
		return contributors[i].Contributions > contributors[j].Contributions
	})

	// Apply limit if specified
	if limit > 0 && limit < len(contributors) {
		contributors = contributors[:limit]
	}

	return contributors, nil
}

func prefetchData(client *github.Client) {
	// Prefetch stats
	go func() {
		stats, err := fetchStats(client)
		if err != nil {
			log.Printf("Error prefetching stats: %v", err)
			return
		}
		cache.updateStats(stats)
	}()

	// Prefetch contributors
	go func() {
		contributors, err := fetchTopContributors(client, 0) // Fetch all contributors
		if err != nil {
			log.Printf("Error prefetching contributors: %v", err)
			return
		}
		cache.updateContributors(contributors)
	}()
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}
}

func main() {
	// Set release mode
	gin.SetMode(gin.ReleaseMode)

	// Create router with trusted proxies
	r := gin.New()
	r.SetTrustedProxies(nil) // Trust all proxies
	r.Use(gin.Recovery())    // Add recovery middleware

	loadEnv()

	// Initialize GitHub client
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	// Prefetch data on startup
	prefetchData(client)

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, githubBaseURL)
	})

	// Repository redirection endpoint
	r.GET("/:repo", func(c *gin.Context) {
		repo := c.Param("repo")

		// Handle special routes
		switch repo {
		case "stats":
			stats, exists := cache.getStats()
			if !exists || cache.isStale(&cache.stats) {
				var err error
				stats, err = fetchStats(client)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				cache.updateStats(stats)
			}

			// Calculate totals
			var totalStars, totalForks, totalContributors, totalCommits int
			for _, repo := range stats {
				totalStars += repo.Stars
				totalForks += repo.Forks
				totalContributors += repo.Contributors
				totalCommits += repo.Commits
			}

			// Sort repositories by stars in descending order
			sort.Slice(stats, func(i, j int) bool {
				return stats[i].Stars > stats[j].Stars
			})

			response := gin.H{
				"repositories":      stats,
				"totalStars":        totalStars,
				"totalForks":        totalForks,
				"totalContributors": totalContributors,
				"totalCommits":      totalCommits,
				"githubUrl":         githubBaseURL,
			}

			c.JSON(http.StatusOK, response)
			return
		case "contributors":
			limitStr := c.DefaultQuery("limit", "0")
			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				limit = 0 // If invalid limit, show all contributors
			}

			contributors, exists := cache.getContributors()
			if !exists || cache.isStale(&cache.contributors) {
				var err error
				contributors, err = fetchTopContributors(client, 0) // Always fetch all for cache
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				cache.updateContributors(contributors)
			}

			// Apply limit to cached data
			if limit > 0 && limit < len(contributors) {
				contributors = contributors[:limit]
			}

			c.JSON(http.StatusOK, contributors)
			return
		}

		// Default case: redirect to GitHub
		redirectURL := githubBaseURL + repo
		c.Redirect(http.StatusMovedPermanently, redirectURL)
	})

	// Start the server
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
