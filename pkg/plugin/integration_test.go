package plugin

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// IntegrationTestSuite represents a complete integration test environment
type IntegrationTestSuite struct {
	pluginManager *PluginManager
	tempDir       string
}

// NewIntegrationTestSuite creates a new integration test suite
func NewIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
	tempDir := t.TempDir()

	return &IntegrationTestSuite{
		pluginManager: NewPluginManager(),
		tempDir:       tempDir,
	}
}

// MockComplexPlugin represents a more complex plugin for integration testing
type MockComplexPlugin struct {
	*MockPlugin
	executionLog []string
	mu           sync.Mutex
}

func NewMockComplexPlugin(name, version string) *MockComplexPlugin {
	return &MockComplexPlugin{
		MockPlugin:   NewMockPlugin(name, version),
		executionLog: make([]string, 0),
	}
}

func (m *MockComplexPlugin) ProcessRequest(ctx context.Context, request interface{}) (interface{}, error) {
	m.mu.Lock()
	m.executionLog = append(m.executionLog, fmt.Sprintf("ProcessRequest: %v", request))
	m.mu.Unlock()

	return fmt.Sprintf("processed-%v", request), nil
}

func (m *MockComplexPlugin) GetExecutionLog() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	log := make([]string, len(m.executionLog))
	copy(log, m.executionLog)
	return log
}

// Mock HTTP server for testing
func createMockServer(t *testing.T) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/simple":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("simple test data"))
		case "/large":
			w.WriteHeader(http.StatusOK)
			// Write 1MB of data
			data := strings.Repeat("a", 1024*1024)
			_, _ = w.Write([]byte(data))
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("server error"))
		case "/auth-required":
			auth := r.Header.Get("Authorization")
			if auth != "Bearer mock-token-123" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("authenticated data"))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
		}
	}))

	t.Cleanup(func() {
		server.Close()
	})

	return server
}

func TestEndToEndWithPlugins(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	server := createMockServer(t)

	t.Run("SimplePluginExecution", func(t *testing.T) {
		// Create and register a complex plugin
		complexPlugin := NewMockComplexPlugin("e2e-plugin", "1.0.0")

		err := suite.pluginManager.Register(complexPlugin)
		if err != nil {
			t.Fatalf("Failed to register plugin: %v", err)
		}

		// Initialize plugin
		err = complexPlugin.Init(map[string]interface{}{
			"test_mode": true,
		})
		if err != nil {
			t.Fatalf("Failed to initialize plugin: %v", err)
		}

		// Simulate plugin processing a request
		ctx := context.Background()
		result, err := complexPlugin.ProcessRequest(ctx, "test-request")
		if err != nil {
			t.Fatalf("Plugin request processing failed: %v", err)
		}

		expected := "processed-test-request"
		if result != expected {
			t.Errorf("Expected '%s', got: %v", expected, result)
		}

		// Verify execution log
		log := complexPlugin.GetExecutionLog()
		if len(log) != 1 {
			t.Errorf("Expected 1 log entry, got: %d", len(log))
		}

		expectedLog := "ProcessRequest: test-request"
		if log[0] != expectedLog {
			t.Errorf("Expected log '%s', got: '%s'", expectedLog, log[0])
		}
	})

	t.Run("AuthenticationFlow", func(t *testing.T) {
		// Create auth plugin
		authPlugin := NewMockAuthPlugin("auth-e2e", "1.0.0")
		err := suite.pluginManager.Register(authPlugin)
		if err != nil {
			t.Fatalf("Failed to register auth plugin: %v", err)
		}

		err = authPlugin.Init(map[string]interface{}{
			"token_endpoint": server.URL + "/token",
		})
		if err != nil {
			t.Fatalf("Failed to initialize auth plugin: %v", err)
		}

		// Test authentication
		ctx := context.Background()
		token, err := authPlugin.Authenticate(ctx, map[string]string{
			"username": "testuser",
			"password": "testpass",
		})
		if err != nil {
			t.Fatalf("Authentication failed: %v", err)
		}

		// Verify token format
		tokenMap, ok := token.(map[string]string)
		if !ok {
			t.Fatal("Expected token to be map[string]string")
		}

		if tokenMap["token"] != "mock-token-123" {
			t.Errorf("Expected token 'mock-token-123', got: %s", tokenMap["token"])
		}

		// Simulate using the token (this would be part of the actual download process)
		// In a real scenario, this token would be used in HTTP headers
		if len(tokenMap["token"]) == 0 {
			t.Error("Token should not be empty")
		}
	})

	t.Run("PluginCleanup", func(t *testing.T) {
		// Register multiple plugins
		plugin1 := NewMockPlugin("cleanup1", "1.0.0")
		plugin2 := NewMockPlugin("cleanup2", "1.0.0")

		_ = suite.pluginManager.Register(plugin1)
		_ = suite.pluginManager.Register(plugin2)

		// Initialize plugins
		_ = plugin1.Init(map[string]interface{}{})
		_ = plugin2.Init(map[string]interface{}{})

		// Verify plugins are initialized
		if !plugin1.IsInitialized() {
			t.Error("plugin1 should be initialized")
		}
		if !plugin2.IsInitialized() {
			t.Error("plugin2 should be initialized")
		}

		// Close plugin manager
		err := suite.pluginManager.Close()
		if err != nil {
			t.Fatalf("Failed to close plugin manager: %v", err)
		}

		// Verify plugins are closed
		if !plugin1.IsClosed() {
			t.Error("plugin1 should be closed")
		}
		if !plugin2.IsClosed() {
			t.Error("plugin2 should be closed")
		}
	})
}

func TestMultiplePluginInteraction(t *testing.T) {
	suite := NewIntegrationTestSuite(t)

	t.Run("PluginChaining", func(t *testing.T) {
		// Create multiple transform plugins that chain together
		plugin1 := NewMockTransformPlugin("transform1", "1.0.0")
		plugin2 := NewMockTransformPlugin("transform2", "1.0.0")
		plugin3 := NewMockTransformPlugin("transform3", "1.0.0")

		// Register plugins
		_ = suite.pluginManager.Register(plugin1)
		_ = suite.pluginManager.Register(plugin2)
		_ = suite.pluginManager.Register(plugin3)

		// Initialize plugins
		_ = plugin1.Init(map[string]interface{}{"stage": "first"})
		_ = plugin2.Init(map[string]interface{}{"stage": "second"})
		_ = plugin3.Init(map[string]interface{}{"stage": "third"})

		// Chain transformations
		ctx := context.Background()
		input := "original-data"

		// First transformation
		result1, err := plugin1.Transform(ctx, input)
		if err != nil {
			t.Fatalf("First transformation failed: %v", err)
		}

		// Second transformation
		result2, err := plugin2.Transform(ctx, result1)
		if err != nil {
			t.Fatalf("Second transformation failed: %v", err)
		}

		// Third transformation
		result3, err := plugin3.Transform(ctx, result2)
		if err != nil {
			t.Fatalf("Third transformation failed: %v", err)
		}

		// Verify chaining worked correctly
		expected := "transformed-transformed-transformed-original-data"
		if result3 != expected {
			t.Errorf("Expected '%s', got: %v", expected, result3)
		}
	})

	t.Run("ConcurrentPluginExecution", func(t *testing.T) {
		// Create multiple plugins for concurrent execution
		numPlugins := 10
		plugins := make([]*MockComplexPlugin, numPlugins)

		for i := 0; i < numPlugins; i++ {
			plugin := NewMockComplexPlugin(fmt.Sprintf("concurrent-%d", i), "1.0.0")
			plugins[i] = plugin
			_ = suite.pluginManager.Register(plugin)
			_ = plugin.Init(map[string]interface{}{"id": i})
		}

		// Execute plugins concurrently
		var wg sync.WaitGroup
		results := make(chan string, numPlugins)
		errors := make(chan error, numPlugins)

		ctx := context.Background()

		for i, plugin := range plugins {
			wg.Add(1)
			go func(p *MockComplexPlugin, id int) {
				defer wg.Done()

				result, err := p.ProcessRequest(ctx, fmt.Sprintf("request-%d", id))
				if err != nil {
					errors <- err
					return
				}

				results <- result.(string)
			}(plugin, i)
		}

		wg.Wait()
		close(results)
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Plugin execution error: %v", err)
		}

		// Verify all results
		resultCount := 0
		for result := range results {
			resultCount++
			if !strings.HasPrefix(result, "processed-request-") {
				t.Errorf("Unexpected result format: %s", result)
			}
		}

		if resultCount != numPlugins {
			t.Errorf("Expected %d results, got: %d", numPlugins, resultCount)
		}

		// Verify each plugin was executed
		for i, plugin := range plugins {
			log := plugin.GetExecutionLog()
			if len(log) != 1 {
				t.Errorf("Plugin %d should have 1 log entry, got: %d", i, len(log))
			}
		}
	})

	t.Run("PluginDependencyOrder", func(t *testing.T) {
		// Test plugins that depend on execution order
		var executionOrder []string
		var mu sync.Mutex

		// Create plugins with hooks that track execution order
		plugin1 := NewMockPlugin("order1", "1.0.0")
		plugin2 := NewMockPlugin("order2", "1.0.0")
		plugin3 := NewMockPlugin("order3", "1.0.0")

		_ = suite.pluginManager.Register(plugin1)
		_ = suite.pluginManager.Register(plugin2)
		_ = suite.pluginManager.Register(plugin3)

		// Add hooks that track execution order
		hook1 := func(data interface{}) error {
			mu.Lock()
			executionOrder = append(executionOrder, "order1")
			mu.Unlock()
			return nil
		}

		hook2 := func(data interface{}) error {
			mu.Lock()
			executionOrder = append(executionOrder, "order2")
			mu.Unlock()
			return nil
		}

		hook3 := func(data interface{}) error {
			mu.Lock()
			executionOrder = append(executionOrder, "order3")
			mu.Unlock()
			return nil
		}

		// Add hooks (they should execute in the order they were added)
		suite.pluginManager.AddHook(PreDownloadHook, hook1)
		suite.pluginManager.AddHook(PreDownloadHook, hook2)
		suite.pluginManager.AddHook(PreDownloadHook, hook3)

		// Execute hooks
		err := suite.pluginManager.ExecuteHook(PreDownloadHook, "test-data")
		if err != nil {
			t.Fatalf("Hook execution failed: %v", err)
		}

		// Verify execution order
		mu.Lock()
		if len(executionOrder) != 3 {
			t.Errorf("Expected 3 hooks executed, got: %d", len(executionOrder))
		}

		expectedOrder := []string{"order1", "order2", "order3"}
		for i, expected := range expectedOrder {
			if i >= len(executionOrder) || executionOrder[i] != expected {
				t.Errorf("Expected execution order %v, got: %v", expectedOrder, executionOrder)
				break
			}
		}
		mu.Unlock()
	})
}

func TestPluginPerformance(t *testing.T) {
	suite := NewIntegrationTestSuite(t)

	t.Run("PluginRegistrationPerformance", func(t *testing.T) {
		start := time.Now()
		numPlugins := 1000

		// Register many plugins
		for i := 0; i < numPlugins; i++ {
			plugin := NewMockPlugin(fmt.Sprintf("perf-%d", i), "1.0.0")
			err := suite.pluginManager.Register(plugin)
			if err != nil {
				t.Fatalf("Failed to register plugin %d: %v", i, err)
			}
		}

		duration := time.Since(start)

		// Should complete within reasonable time (1 second for 1000 plugins)
		if duration > time.Second {
			t.Errorf("Plugin registration took too long: %v", duration)
		}

		// Verify all plugins are registered
		plugins := suite.pluginManager.ListPlugins()
		if len(plugins) != numPlugins {
			t.Errorf("Expected %d plugins, got: %d", numPlugins, len(plugins))
		}

		t.Logf("Registered %d plugins in %v (%.2f plugins/ms)",
			numPlugins, duration, float64(numPlugins)/float64(duration.Milliseconds()))
	})

	t.Run("PluginExecutionPerformance", func(t *testing.T) {
		// Create plugin for performance testing
		plugin := NewMockTransformPlugin("perf-exec", "1.0.0")
		_ = suite.pluginManager.Register(plugin)
		_ = plugin.Init(map[string]interface{}{})

		ctx := context.Background()
		numExecutions := 10000

		start := time.Now()

		// Execute plugin many times
		for i := 0; i < numExecutions; i++ {
			_, err := plugin.Transform(ctx, fmt.Sprintf("data-%d", i))
			if err != nil {
				t.Fatalf("Plugin execution failed at iteration %d: %v", i, err)
			}
		}

		duration := time.Since(start)

		// Should complete within reasonable time
		if duration > 5*time.Second {
			t.Errorf("Plugin execution took too long: %v", duration)
		}

		execsPerSecond := float64(numExecutions) / duration.Seconds()
		t.Logf("Executed plugin %d times in %v (%.2f executions/second)",
			numExecutions, duration, execsPerSecond)

		// Should achieve reasonable throughput (at least 1000 executions/second)
		if execsPerSecond < 1000 {
			t.Errorf("Plugin execution too slow: %.2f executions/second", execsPerSecond)
		}
	})

	t.Run("HookExecutionPerformance", func(t *testing.T) {
		numHooks := 100
		executionCounter := int64(0)
		var mu sync.Mutex

		// Add many hooks
		for i := 0; i < numHooks; i++ {
			hookFunc := func(data interface{}) error {
				mu.Lock()
				executionCounter++
				mu.Unlock()
				return nil
			}
			suite.pluginManager.AddHook(PostDownloadHook, hookFunc)
		}

		numExecutions := 1000
		start := time.Now()

		// Execute hooks many times
		for i := 0; i < numExecutions; i++ {
			err := suite.pluginManager.ExecuteHook(PostDownloadHook, fmt.Sprintf("data-%d", i))
			if err != nil {
				t.Fatalf("Hook execution failed at iteration %d: %v", i, err)
			}
		}

		duration := time.Since(start)

		// Verify all hooks were executed
		mu.Lock()
		expectedExecutions := int64(numHooks * numExecutions)
		if executionCounter != expectedExecutions {
			t.Errorf("Expected %d hook executions, got: %d", expectedExecutions, executionCounter)
		}
		mu.Unlock()

		hookExecsPerSecond := float64(expectedExecutions) / duration.Seconds()
		t.Logf("Executed %d hooks %d times in %v (%.2f hook-executions/second)",
			numHooks, numExecutions, duration, hookExecsPerSecond)

		// Should complete within reasonable time
		if duration > 10*time.Second {
			t.Errorf("Hook execution took too long: %v", duration)
		}
	})

	t.Run("MemoryUsage", func(t *testing.T) {
		// This test ensures plugins don't cause memory leaks
		initialPlugins := len(suite.pluginManager.ListPlugins())

		// Create and destroy many plugins
		for iteration := 0; iteration < 10; iteration++ {
			plugins := make([]*MockPlugin, 100)

			// Create plugins
			for i := 0; i < 100; i++ {
				plugin := NewMockPlugin(fmt.Sprintf("memory-test-%d-%d", iteration, i), "1.0.0")
				plugins[i] = plugin
				_ = suite.pluginManager.Register(plugin)
				_ = plugin.Init(map[string]interface{}{"iteration": iteration})
			}

			// Use plugins - just call a basic method
			for _, plugin := range plugins {
				plugin.Name() // Just call a basic method to simulate usage
			}

			// Remove plugins
			for _, plugin := range plugins {
				_ = suite.pluginManager.Unregister(plugin.Name())
			}

			// Verify plugins are cleaned up
			currentPlugins := len(suite.pluginManager.ListPlugins())
			if currentPlugins != initialPlugins {
				t.Errorf("Memory leak detected: expected %d plugins, got %d after iteration %d",
					initialPlugins, currentPlugins, iteration)
			}
		}
	})
}
