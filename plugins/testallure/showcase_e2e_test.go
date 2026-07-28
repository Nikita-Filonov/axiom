package testallure_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testallure"
	"github.com/stretchr/testify/require"
)

const showcaseResultsEnv = "AXIOM_ALLURE_SHOWCASE_DIR"

func TestPlugin_GeneratesShowcaseAllureResults(t *testing.T) {
	resultsDir := os.Getenv(showcaseResultsEnv)
	if resultsDir == "" {
		t.Skipf("set %s to generate persistent showcase results", showcaseResultsEnv)
	}
	require.Equal(t, resultsDir, os.Getenv("ALLURE_RESULTS_DIR"))
	require.NoError(t, os.MkdirAll(resultsDir, 0o755))
	writeShowcaseMetadata(t, resultsDir)

	runner := axiom.NewRunner(
		axiom.WithRunnerParallel(axiom.WithParallelEnabled()),
		axiom.WithRunnerPlugins(testallure.Plugin()),
	)

	runner.RunCase(t, loginCase(), runLoginCase)
	runner.RunCase(t, declinedPaymentCase(), runDeclinedPaymentCase)
	runner.RunCase(t, productSearchCase(), runProductSearchCase)
	runner.RunCase(t, disabledFeatureCase(), runDisabledFeatureCase)
	runner.RunCase(t, inventoryMismatchCase(), runInventoryMismatchCase)
	runner.RunCase(t, brokenWebhookCase(), runBrokenWebhookCase)
}

func loginCase() axiom.Case {
	return axiom.NewCase(
		axiom.WithCaseID("AUTH-42"),
		axiom.WithCaseName("user can login with valid credentials"),
		axiom.WithCaseDescription(
			"Authenticates an active user and verifies that a reusable session is returned.",
		),
		axiom.WithCaseMeta(
			axiom.WithMetaParentSuite("Axiom showcase"),
			axiom.WithMetaSuite("Public API"),
			axiom.WithMetaSubSuite("Authentication"),
			axiom.WithMetaEpic("Identity and access"),
			axiom.WithMetaFeature("Login"),
			axiom.WithMetaStory("Valid credentials"),
			axiom.WithMetaSeverity(axiom.SeverityCritical),
			axiom.WithMetaLayer("api"),
			axiom.WithMetaPlatform("staging"),
			axiom.WithMetaTags("smoke", "regression"),
			axiom.WithMetaLabel("owner", "identity-team"),
			axiom.WithMetaLabel("component", "auth-service"),
			axiom.WithMetaIssue("AXIOM-42"),
			axiom.WithMetaTestCase("USERS-001"),
		),
	)
}

func runLoginCase(cfg *axiom.Config) {
	cfg.Setup("seed active user", func() {
		cfg.Artefact(axiom.NewTextArtefact(
			"setup.log",
			"created user alice@example.test\nassigned role: customer",
		))
	})

	cfg.Step("prepare login request", func() {
		cfg.Artefact(mustJSONArtefact(cfg, "request.json", map[string]any{
			"email":    "alice@example.test",
			"password": "<redacted>",
		}))
	})

	cfg.Step("POST /v1/sessions", func() {
		cfg.Artefact(axiom.NewTextArtefact(
			"http-trace.txt",
			"POST /v1/sessions HTTP/1.1\ncontent-type: application/json\n\nHTTP/1.1 201 Created",
		))

		cfg.Step("validate response contract", func() {
			cfg.Artefact(mustJSONArtefact(cfg, "response.json", map[string]any{
				"sessionId": "ses_showcase_001",
				"expiresIn": 3600,
				"user": map[string]any{
					"id":    "usr_001",
					"email": "alice@example.test",
				},
			}))
		})

		cfg.Step("verify session can access profile", func() {
			cfg.Artefact(axiom.NewTextArtefact(
				"verification.txt",
				"GET /v1/profile -> 200 OK",
			))
		})
	})

	cfg.Teardown("delete seeded user", func() {
		cfg.Artefact(axiom.NewTextArtefact("cleanup.log", "deleted usr_001"))
	})
}

func declinedPaymentCase() axiom.Case {
	return axiom.NewCase(
		axiom.WithCaseID("PAY-17"),
		axiom.WithCaseName("declined card does not create an order"),
		axiom.WithCaseDescription(
			"Verifies payment decline handling and ensures inventory remains unchanged.",
		),
		axiom.WithCaseMeta(
			axiom.WithMetaParentSuite("Axiom showcase"),
			axiom.WithMetaSuite("Public API"),
			axiom.WithMetaSubSuite("Checkout"),
			axiom.WithMetaEpic("Commerce"),
			axiom.WithMetaFeature("Card payments"),
			axiom.WithMetaStory("Declined payment"),
			axiom.WithMetaSeverity(axiom.SeverityBlocker),
			axiom.WithMetaLayer("api"),
			axiom.WithMetaPlatform("staging"),
			axiom.WithMetaTags("payments", "regression"),
			axiom.WithMetaLabel("owner", "checkout-team"),
			axiom.WithMetaLabel("component", "payment-orchestrator"),
			axiom.WithMetaIssue("PAY-917"),
			axiom.WithMetaTestCase("CHECKOUT-204"),
		),
	)
}

func runDeclinedPaymentCase(cfg *axiom.Config) {
	cfg.Setup("reserve inventory", func() {
		cfg.Artefact(mustJSONArtefact(cfg, "reservation.json", map[string]any{
			"sku":      "SKU-RED-42",
			"quantity": 1,
			"state":    "reserved",
		}))
	})

	cfg.Step("create checkout", func() {
		cfg.Artefact(mustJSONArtefact(cfg, "checkout.json", map[string]any{
			"currency": "USD",
			"total":    12999,
			"items":    []string{"SKU-RED-42"},
		}))

		cfg.Step("submit declined card", func() {
			cfg.Artefact(axiom.NewTextArtefact(
				"gateway-response.txt",
				"provider=sandbox-pay\ncode=card_declined\nreason=insufficient_funds",
			))
		})

		cfg.Step("verify checkout state", func() {
			cfg.Artefact(mustJSONArtefact(cfg, "checkout-state.json", map[string]any{
				"payment": "declined",
				"orderId": nil,
			}))
		})
	})

	cfg.Teardown("release inventory", func() {
		cfg.Artefact(axiom.NewTextArtefact("cleanup.log", "released SKU-RED-42"))
	})
}

func productSearchCase() axiom.Case {
	return axiom.NewCase(
		axiom.WithCaseID("CAT-8"),
		axiom.WithCaseName("product search combines filters and sorting"),
		axiom.WithCaseDescription(
			"Searches the catalog by query, availability and price, then validates ranking.",
		),
		axiom.WithCaseMeta(
			axiom.WithMetaParentSuite("Axiom showcase"),
			axiom.WithMetaSuite("Web storefront"),
			axiom.WithMetaSubSuite("Catalog"),
			axiom.WithMetaEpic("Discovery"),
			axiom.WithMetaFeature("Product search"),
			axiom.WithMetaStory("Filtered search"),
			axiom.WithMetaSeverity(axiom.SeverityNormal),
			axiom.WithMetaLayer("e2e"),
			axiom.WithMetaPlatform("chromium"),
			axiom.WithMetaTags("search", "web"),
			axiom.WithMetaLabel("owner", "catalog-team"),
			axiom.WithMetaLabel("component", "search-api"),
			axiom.WithMetaTestCase("CATALOG-088"),
		),
	)
}

func runProductSearchCase(cfg *axiom.Config) {
	cfg.Setup("open storefront", func() {
		cfg.Artefact(axiom.NewTextArtefact(
			"browser.txt",
			"browser=chromium\nviewport=1440x900\nbaseURL=https://shop.example.test",
		))
	})

	cfg.Step("enter search query", func() {
		cfg.Artefact(mustJSONArtefact(cfg, "search-request.json", map[string]any{
			"query":       "running shoes",
			"inStockOnly": true,
			"maxPrice":    150,
			"sort":        "relevance",
		}))
	})

	cfg.Step("validate ranked results", func() {
		cfg.Artefact(mustJSONArtefact(cfg, "search-response.json", map[string]any{
			"total": 3,
			"items": []map[string]any{
				{"sku": "RUN-001", "score": 0.98},
				{"sku": "RUN-014", "score": 0.91},
				{"sku": "RUN-022", "score": 0.87},
			},
		}))

		cfg.Step("verify every result is available", func() {})
		cfg.Step("verify relevance is descending", func() {})
	})

	cfg.Teardown("close browser", func() {})
}

func disabledFeatureCase() axiom.Case {
	return axiom.NewCase(
		axiom.WithCaseID("EXP-3"),
		axiom.WithCaseName("AI recommendations improve product discovery"),
		axiom.WithCaseDescription(
			"Showcase of a skipped result when an experimental feature is disabled.",
		),
		axiom.WithCaseMeta(
			axiom.WithMetaParentSuite("Axiom showcase"),
			axiom.WithMetaSuite("Experimental"),
			axiom.WithMetaEpic("Discovery"),
			axiom.WithMetaFeature("AI recommendations"),
			axiom.WithMetaStory("Personalized ranking"),
			axiom.WithMetaSeverity(axiom.SeverityMinor),
			axiom.WithMetaTags("experimental", "feature-flag"),
			axiom.WithMetaLabel("owner", "recommendations-team"),
		),
	)
}

func runDisabledFeatureCase(cfg *axiom.Config) {
	cfg.Step("read feature flag", func() {
		cfg.Artefact(axiom.NewTextArtefact(
			"feature-flag.txt",
			"ai_recommendations=false",
		))
		cfg.SubT.Skip("AI recommendations are disabled in the showcase environment")
	})
}

func inventoryMismatchCase() axiom.Case {
	return axiom.NewCase(
		axiom.WithCaseID("INV-51"),
		axiom.WithCaseName("inventory quantity is updated after reservation"),
		axiom.WithCaseDescription(
			"Intentionally failed showcase case with an assertion-style mismatch.",
		),
		axiom.WithCaseMeta(
			axiom.WithMetaParentSuite("Axiom showcase"),
			axiom.WithMetaSuite("Public API"),
			axiom.WithMetaSubSuite("Inventory"),
			axiom.WithMetaEpic("Commerce"),
			axiom.WithMetaFeature("Stock reservation"),
			axiom.WithMetaStory("Available quantity"),
			axiom.WithMetaSeverity(axiom.SeverityCritical),
			axiom.WithMetaLayer("api"),
			axiom.WithMetaPlatform("staging"),
			axiom.WithMetaTags("inventory", "negative-showcase"),
			axiom.WithMetaLabel("owner", "fulfillment-team"),
			axiom.WithMetaLabel("component", "inventory-service"),
			axiom.WithMetaIssue("INV-551"),
			axiom.WithMetaTestCase("STOCK-031"),
		),
	)
}

func runInventoryMismatchCase(cfg *axiom.Config) {
	cfg.Setup("create stock item", func() {
		cfg.Artefact(mustJSONArtefact(cfg, "initial-stock.json", map[string]any{
			"sku":       "SKU-BLUE-09",
			"available": 10,
			"reserved":  0,
		}))
	})

	cfg.Step("reserve three units", func() {
		cfg.Artefact(mustJSONArtefact(cfg, "reservation-request.json", map[string]any{
			"sku":      "SKU-BLUE-09",
			"quantity": 3,
		}))
	})

	cfg.Step("validate available quantity", func() {
		cfg.Artefact(mustJSONArtefact(cfg, "actual-stock.json", map[string]any{
			"sku":       "SKU-BLUE-09",
			"available": 8,
			"reserved":  2,
		}))
		cfg.SubT.Errorf("available quantity mismatch: expected 7, got 8")
	})

	cfg.Teardown("delete stock item", func() {
		cfg.Artefact(axiom.NewTextArtefact("cleanup.log", "deleted SKU-BLUE-09"))
	})
}

func brokenWebhookCase() axiom.Case {
	return axiom.NewCase(
		axiom.WithCaseID("HOOK-13"),
		axiom.WithCaseName("payment webhook updates order asynchronously"),
		axiom.WithCaseDescription(
			"Intentionally broken showcase case that panics after receiving malformed data.",
		),
		axiom.WithCaseMeta(
			axiom.WithMetaParentSuite("Axiom showcase"),
			axiom.WithMetaSuite("Event processing"),
			axiom.WithMetaSubSuite("Payment webhooks"),
			axiom.WithMetaEpic("Commerce"),
			axiom.WithMetaFeature("Webhook processing"),
			axiom.WithMetaStory("Payment confirmation"),
			axiom.WithMetaSeverity(axiom.SeverityBlocker),
			axiom.WithMetaLayer("integration"),
			axiom.WithMetaPlatform("staging"),
			axiom.WithMetaTags("webhook", "negative-showcase"),
			axiom.WithMetaLabel("owner", "platform-team"),
			axiom.WithMetaLabel("component", "event-consumer"),
			axiom.WithMetaIssue("HOOK-313"),
			axiom.WithMetaTestCase("EVENTS-019"),
		),
	)
}

func runBrokenWebhookCase(cfg *axiom.Config) {
	cfg.Setup("publish payment event", func() {
		cfg.Artefact(mustJSONArtefact(cfg, "webhook.json", map[string]any{
			"event":   "payment.succeeded",
			"orderId": "ord_404",
			"payload": nil,
		}))
	})

	cfg.Step("wait for event consumer", func() {
		cfg.Artefact(axiom.NewTextArtefact(
			"consumer.log",
			"received payment.succeeded for ord_404\nprocessing payload",
		))
	})

	panic("decode payment webhook: unexpected end of JSON input")
}

func mustJSONArtefact(cfg *axiom.Config, name string, value any) axiom.Artefact {
	cfg.SubT.Helper()

	artefact, err := axiom.NewJSONArtefact(name, value)
	require.NoError(cfg.SubT, err)
	return artefact
}

func writeShowcaseMetadata(t *testing.T, resultsDir string) {
	t.Helper()

	environment := []byte(
		"Application=Axiom demo shop\n" +
			"Environment=showcase\n" +
			"Go=1.25.5\n" +
			"Allure.Adapter=allure-framework/allure-go v1.2.1\n",
	)
	require.NoError(
		t,
		os.WriteFile(filepath.Join(resultsDir, "environment.properties"), environment, 0o644),
	)

	executor := []byte(`{
  "name": "Local Axiom showcase",
  "type": "local",
  "buildName": "testallure v0.20.0",
  "buildOrder": 1,
  "reportName": "Axiom testallure showcase"
}`)
	require.NoError(
		t,
		os.WriteFile(filepath.Join(resultsDir, "executor.json"), executor, 0o644),
	)
}
