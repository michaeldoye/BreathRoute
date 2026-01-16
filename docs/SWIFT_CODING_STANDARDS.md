# Swift Coding Standards

This document defines the coding standards and best practices for Swift/iOS development in the BreatheRoute project. These standards are based on Apple's Swift API Design Guidelines, industry best practices, and SwiftUI patterns.

## Table of Contents

1. [Formatting and Style](#formatting-and-style)
2. [Naming Conventions](#naming-conventions)
3. [SwiftUI Best Practices](#swiftui-best-practices)
4. [Architecture](#architecture)
5. [Error Handling](#error-handling)
6. [Concurrency](#concurrency)
7. [Memory Management](#memory-management)
8. [Testing](#testing)
9. [Security](#security)
10. [Performance](#performance)
11. [Accessibility](#accessibility)
12. [Project Structure](#project-structure)
13. [Dependencies](#dependencies)
14. [Code Review Checklist](#code-review-checklist)

---

## Formatting and Style

### Use SwiftLint

All code must pass SwiftLint checks before merge:

```bash
# Install SwiftLint
brew install swiftlint

# Run linter
swiftlint

# Auto-fix violations where possible
swiftlint --fix
```

### Indentation and Spacing

- Use 4 spaces for indentation (no tabs)
- Maximum line length: 120 characters
- Use blank lines to separate logical sections

```swift
// Good: Clear spacing and structure
final class RouteViewModel: ObservableObject {

    // MARK: - Published Properties

    @Published private(set) var routes: [Route] = []
    @Published private(set) var isLoading = false
    @Published private(set) var error: Error?

    // MARK: - Dependencies

    private let routeService: RouteServiceProtocol
    private let locationManager: LocationManager

    // MARK: - Initialization

    init(
        routeService: RouteServiceProtocol,
        locationManager: LocationManager
    ) {
        self.routeService = routeService
        self.locationManager = locationManager
    }

    // MARK: - Public Methods

    func fetchRoutes() async {
        // Implementation
    }
}
```

### Braces and Control Flow

```swift
// Good: Opening brace on same line
if condition {
    doSomething()
} else {
    doSomethingElse()
}

// Good: Single-line guard for simple checks
guard let user = user else { return }

// Good: Multi-line guard for complex checks
guard
    let user = user,
    let token = token,
    token.isValid
else {
    throw AuthError.invalidToken
}

// Good: Switch statements
switch status {
case .loading:
    showSpinner()
case .success(let data):
    display(data)
case .failure(let error):
    showError(error)
}
```

### Trailing Closures

```swift
// Good: Single trailing closure
routes.filter { $0.exposureScore < 50 }

// Good: Multiple closures - use labeled syntax
UIView.animate(
    withDuration: 0.3,
    animations: {
        view.alpha = 1
    },
    completion: { finished in
        print("Animation completed")
    }
)

// Good: SwiftUI modifiers
Button("Calculate Route") {
    viewModel.calculateRoute()
}
.buttonStyle(.borderedProminent)
.disabled(viewModel.isLoading)
```

### Type Inference

```swift
// Good: Let Swift infer when obvious
let name = "BreatheRoute"
let count = 42
let routes: [Route] = []

// Good: Explicit when clarity helps
let score: Double = 0
let options: [String: Any] = [:]

// Avoid: Redundant type annotation
let name: String = "BreatheRoute"  // String is obvious
```

---

## Naming Conventions

### General Rules

- Use camelCase for variables, functions, and properties
- Use PascalCase for types and protocols
- Be descriptive but concise
- Avoid abbreviations except common ones (URL, ID, etc.)

### Variables and Properties

```swift
// Good: Clear, descriptive names
let maximumExposureScore = 100.0
var currentUserLocation: CLLocationCoordinate2D?
var isAuthenticated = false
var hasCompletedOnboarding = false

// Good: Use of common abbreviations
let userID: String
let apiURL: URL
let jsonData: Data

// Avoid: Unclear abbreviations
let maxExpScr = 100.0  // What is this?
let usrLoc: CLLocationCoordinate2D?  // Unclear
```

### Functions and Methods

```swift
// Good: Verb phrases for actions
func fetchRoutes() async throws -> [Route]
func calculateExposure(for route: Route) -> Double
func save(_ commute: Commute) throws

// Good: Noun phrases for getters
func distance(from origin: Coordinate, to destination: Coordinate) -> Double

// Good: Boolean prefixes
func contains(_ element: Element) -> Bool
func isValid() -> Bool
func canSubmit() -> Bool

// Good: Mutating vs non-mutating
mutating func sort()       // Mutates in place
func sorted() -> [Element] // Returns new collection

// Good: Factory methods
static func make(with configuration: Configuration) -> Client
class func shared() -> Manager

// Argument labels should read as English
// "Send the message to the recipient"
func send(_ message: Message, to recipient: User)

// "Fade from the first color to the second"
func fade(from firstColor: Color, to secondColor: Color)
```

### Types and Protocols

```swift
// Good: Nouns for types
struct User { }
class RouteCalculator { }
enum AlertSeverity { }

// Good: Capability protocols end in -able, -ible, or -ing
protocol Equatable { }
protocol Identifiable { }
protocol Loading { }

// Good: Other protocols describe what they are
protocol RouteService { }
protocol AuthenticationProvider { }
protocol CommuteRepository { }

// Good: Associated types in protocols
protocol Container {
    associatedtype Element
    func append(_ element: Element)
}
```

### Enums

```swift
// Good: Singular name, lowercase cases
enum RouteMode {
    case walk
    case bike
    case transit
}

enum NetworkError: Error {
    case noConnection
    case timeout
    case invalidResponse(statusCode: Int)
    case decodingFailed(underlying: Error)
}

// Good: Use when raw values are needed
enum APIEndpoint: String {
    case routes = "/v1/routes"
    case commutes = "/v1/commutes"
    case alerts = "/v1/alerts"
}

// Access cases without type prefix when context is clear
let mode: RouteMode = .bike
```

### Constants

```swift
// Good: Group related constants in enums or structs
enum Constants {
    enum API {
        static let baseURL = "https://api.breatheroute.nl"
        static let timeout: TimeInterval = 30
    }

    enum UI {
        static let cornerRadius: CGFloat = 8
        static let animationDuration: TimeInterval = 0.3
    }

    enum Limits {
        static let maxCommutes = 10
        static let maxFreeRoutes = 5
    }
}

// Avoid: Top-level global constants
let kAPIBaseURL = "..."  // Don't use k prefix
let MAX_COMMUTES = 10    // Don't use SCREAMING_CASE
```

---

## SwiftUI Best Practices

### View Composition

```swift
// Good: Small, focused views
struct RouteCard: View {
    let route: Route

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            RouteHeader(mode: route.mode, duration: route.duration)
            ExposureIndicator(score: route.exposureScore)
            RouteDetails(distance: route.distance, segments: route.segments)
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(12)
    }
}

// Good: Extract complex logic into computed properties
struct CommuteRow: View {
    let commute: Commute

    private var formattedSchedule: String {
        commute.schedule.formatted()
    }

    private var statusColor: Color {
        commute.hasActiveAlert ? .orange : .green
    }

    var body: some View {
        HStack {
            Text(commute.label)
            Spacer()
            Text(formattedSchedule)
                .foregroundColor(statusColor)
        }
    }
}
```

### State Management

```swift
// Use appropriate property wrappers
struct ContentView: View {
    // Local state owned by this view
    @State private var searchText = ""

    // Observable object from environment
    @EnvironmentObject private var authManager: AuthManager

    // Observable object passed as dependency
    @StateObject private var viewModel: RouteViewModel

    // Observed object passed from parent
    @ObservedObject var settings: UserSettings

    // Value from environment
    @Environment(\.colorScheme) private var colorScheme

    // Binding to parent's state
    @Binding var isPresented: Bool

    var body: some View {
        // ...
    }
}

// Good: Initialize StateObject with factory
struct CommuteListView: View {
    @StateObject private var viewModel: CommuteViewModel

    init(commuteService: CommuteServiceProtocol) {
        _viewModel = StateObject(
            wrappedValue: CommuteViewModel(service: commuteService)
        )
    }
}
```

### View Modifiers

```swift
// Good: Custom view modifiers for reusable styling
struct CardStyle: ViewModifier {
    func body(content: Content) -> some View {
        content
            .padding()
            .background(Color(.systemBackground))
            .cornerRadius(12)
            .shadow(color: .black.opacity(0.1), radius: 4, y: 2)
    }
}

extension View {
    func cardStyle() -> some View {
        modifier(CardStyle())
    }
}

// Usage
RouteCard(route: route)
    .cardStyle()

// Good: Conditional modifiers
extension View {
    @ViewBuilder
    func `if`<Transform: View>(
        _ condition: Bool,
        transform: (Self) -> Transform
    ) -> some View {
        if condition {
            transform(self)
        } else {
            self
        }
    }
}

// Usage
Text("Route")
    .if(isHighlighted) { $0.bold() }
```

### Lists and ForEach

```swift
// Good: Use Identifiable conformance
struct Route: Identifiable {
    let id: UUID
    // ...
}

// Good: Simple list
List(routes) { route in
    RouteRow(route: route)
}

// Good: List with sections
List {
    Section("Favorites") {
        ForEach(favoriteRoutes) { route in
            RouteRow(route: route)
        }
    }

    Section("Recent") {
        ForEach(recentRoutes) { route in
            RouteRow(route: route)
        }
    }
}

// Avoid: Using indices when not needed
ForEach(0..<routes.count, id: \.self) { index in  // Avoid
    RouteRow(route: routes[index])
}
```

### Navigation

```swift
// Good: NavigationStack with path (iOS 16+)
struct MainView: View {
    @State private var navigationPath = NavigationPath()

    var body: some View {
        NavigationStack(path: $navigationPath) {
            CommuteListView()
                .navigationDestination(for: Commute.self) { commute in
                    CommuteDetailView(commute: commute)
                }
                .navigationDestination(for: Route.self) { route in
                    RouteDetailView(route: route)
                }
        }
    }
}

// Good: Programmatic navigation
Button("View Details") {
    navigationPath.append(selectedCommute)
}
```

---

## Architecture

### MVVM Pattern

```swift
// Model - Plain data structures
struct Commute: Identifiable, Codable {
    let id: UUID
    var label: String
    var origin: Coordinate
    var destination: Coordinate
    var schedule: Schedule
    var alertsEnabled: Bool
}

// ViewModel - Business logic and state
@MainActor
final class CommuteListViewModel: ObservableObject {

    // MARK: - Published State

    @Published private(set) var commutes: [Commute] = []
    @Published private(set) var isLoading = false
    @Published private(set) var error: CommuteError?

    // MARK: - Dependencies

    private let commuteService: CommuteServiceProtocol
    private let analytics: AnalyticsProtocol

    // MARK: - Initialization

    init(
        commuteService: CommuteServiceProtocol,
        analytics: AnalyticsProtocol = Analytics.shared
    ) {
        self.commuteService = commuteService
        self.analytics = analytics
    }

    // MARK: - Public Methods

    func loadCommutes() async {
        isLoading = true
        error = nil

        do {
            commutes = try await commuteService.fetchCommutes()
            analytics.track(.commutesLoaded(count: commutes.count))
        } catch {
            self.error = .loadFailed(error)
            analytics.track(.error(error))
        }

        isLoading = false
    }

    func deleteCommute(_ commute: Commute) async {
        do {
            try await commuteService.delete(commute)
            commutes.removeAll { $0.id == commute.id }
        } catch {
            self.error = .deleteFailed(error)
        }
    }
}

// View - UI only
struct CommuteListView: View {
    @StateObject private var viewModel: CommuteListViewModel

    var body: some View {
        Group {
            if viewModel.isLoading {
                ProgressView()
            } else if let error = viewModel.error {
                ErrorView(error: error, retryAction: loadCommutes)
            } else {
                commuteList
            }
        }
        .task {
            await viewModel.loadCommutes()
        }
    }

    private var commuteList: some View {
        List(viewModel.commutes) { commute in
            CommuteRow(commute: commute)
        }
    }

    private func loadCommutes() {
        Task {
            await viewModel.loadCommutes()
        }
    }
}
```

### Dependency Injection

```swift
// Protocol-based dependencies
protocol CommuteServiceProtocol {
    func fetchCommutes() async throws -> [Commute]
    func create(_ commute: Commute) async throws -> Commute
    func update(_ commute: Commute) async throws -> Commute
    func delete(_ commute: Commute) async throws
}

// Production implementation
final class CommuteService: CommuteServiceProtocol {
    private let apiClient: APIClient

    init(apiClient: APIClient) {
        self.apiClient = apiClient
    }

    func fetchCommutes() async throws -> [Commute] {
        try await apiClient.request(.getCommutes)
    }
}

// Mock for testing
final class MockCommuteService: CommuteServiceProtocol {
    var commutesToReturn: [Commute] = []
    var errorToThrow: Error?
    var createCalled = false

    func fetchCommutes() async throws -> [Commute] {
        if let error = errorToThrow { throw error }
        return commutesToReturn
    }

    func create(_ commute: Commute) async throws -> Commute {
        createCalled = true
        return commute
    }
}

// Dependency container
@MainActor
final class DependencyContainer: ObservableObject {
    let apiClient: APIClient
    let commuteService: CommuteServiceProtocol
    let routeService: RouteServiceProtocol
    let authManager: AuthManager

    init(configuration: Configuration = .production) {
        self.apiClient = APIClient(baseURL: configuration.apiBaseURL)
        self.commuteService = CommuteService(apiClient: apiClient)
        self.routeService = RouteService(apiClient: apiClient)
        self.authManager = AuthManager(apiClient: apiClient)
    }

    // For testing
    init(
        commuteService: CommuteServiceProtocol,
        routeService: RouteServiceProtocol,
        authManager: AuthManager
    ) {
        self.apiClient = APIClient(baseURL: .testing)
        self.commuteService = commuteService
        self.routeService = routeService
        self.authManager = authManager
    }
}
```

### Repository Pattern

```swift
// Repository protocol
protocol CommuteRepositoryProtocol {
    func getAll() async throws -> [Commute]
    func get(id: UUID) async throws -> Commute?
    func save(_ commute: Commute) async throws
    func delete(id: UUID) async throws
}

// Implementation with caching
final class CommuteRepository: CommuteRepositoryProtocol {
    private let remoteDataSource: CommuteRemoteDataSource
    private let localDataSource: CommuteLocalDataSource
    private let cachePolicy: CachePolicy

    init(
        remoteDataSource: CommuteRemoteDataSource,
        localDataSource: CommuteLocalDataSource,
        cachePolicy: CachePolicy = .standard
    ) {
        self.remoteDataSource = remoteDataSource
        self.localDataSource = localDataSource
        self.cachePolicy = cachePolicy
    }

    func getAll() async throws -> [Commute] {
        // Try cache first
        if let cached = try await localDataSource.getAll(),
           !cachePolicy.isExpired(for: cached) {
            return cached
        }

        // Fetch from remote
        let commutes = try await remoteDataSource.fetchAll()

        // Update cache
        try await localDataSource.save(commutes)

        return commutes
    }
}
```

---

## Error Handling

### Error Types

```swift
// Domain-specific errors
enum CommuteError: LocalizedError {
    case notFound(id: UUID)
    case limitExceeded(current: Int, maximum: Int)
    case invalidSchedule(reason: String)
    case networkError(underlying: Error)
    case unauthorized

    var errorDescription: String? {
        switch self {
        case .notFound(let id):
            return "Commute \(id) was not found"
        case .limitExceeded(let current, let maximum):
            return "Cannot create more commutes. You have \(current) of \(maximum) allowed."
        case .invalidSchedule(let reason):
            return "Invalid schedule: \(reason)"
        case .networkError:
            return "A network error occurred. Please try again."
        case .unauthorized:
            return "Please sign in to continue."
        }
    }

    var recoverySuggestion: String? {
        switch self {
        case .networkError:
            return "Check your internet connection and try again."
        case .unauthorized:
            return "Tap Sign In to authenticate."
        default:
            return nil
        }
    }
}
```

### Result Type

```swift
// Use Result for synchronous operations
func validate(_ input: String) -> Result<ValidatedInput, ValidationError> {
    guard !input.isEmpty else {
        return .failure(.empty)
    }
    guard input.count <= 100 else {
        return .failure(.tooLong(maxLength: 100))
    }
    return .success(ValidatedInput(value: input))
}

// Handle results
switch validate(userInput) {
case .success(let validated):
    save(validated)
case .failure(let error):
    showError(error)
}
```

### Throwing Functions

```swift
// Prefer async throws for async operations
func fetchRoute(from origin: Coordinate, to destination: Coordinate) async throws -> Route {
    let request = RouteRequest(origin: origin, destination: destination)

    guard let response = try await apiClient.send(request) else {
        throw RouteError.noRouteFound
    }

    return try Route(from: response)
}

// Handle errors with do-catch
func loadRoute() async {
    do {
        let route = try await routeService.fetchRoute(from: origin, to: destination)
        self.route = route
    } catch let error as RouteError {
        self.error = error
    } catch {
        self.error = .unknown(underlying: error)
    }
}
```

### Optional Handling

```swift
// Good: Guard for early exit
func processUser(_ user: User?) {
    guard let user = user else {
        return
    }
    // Use user safely
}

// Good: If-let for conditional logic
if let email = user.email {
    sendEmail(to: email)
}

// Good: Optional chaining
let city = user?.address?.city

// Good: Nil coalescing
let name = user?.name ?? "Anonymous"

// Good: Map and flatMap
let uppercasedName = user?.name.map { $0.uppercased() }
let emailDomain = user?.email.flatMap { $0.split(separator: "@").last }

// Avoid: Force unwrapping
let name = user!.name  // Crash if nil

// Avoid: Implicit unwrapping in most cases
var user: User!  // Usually wrong
```

---

## Concurrency

### Async/Await

```swift
// Good: Async function
func fetchUserData() async throws -> UserData {
    async let profile = fetchProfile()
    async let settings = fetchSettings()
    async let commutes = fetchCommutes()

    return try await UserData(
        profile: profile,
        settings: settings,
        commutes: commutes
    )
}

// Good: Task for launching async work from sync context
func refreshData() {
    Task {
        do {
            try await viewModel.refresh()
        } catch {
            showError(error)
        }
    }
}

// Good: TaskGroup for dynamic concurrent work
func fetchAllRoutes(for commutes: [Commute]) async throws -> [Route] {
    try await withThrowingTaskGroup(of: Route.self) { group in
        for commute in commutes {
            group.addTask {
                try await self.fetchRoute(for: commute)
            }
        }

        var routes: [Route] = []
        for try await route in group {
            routes.append(route)
        }
        return routes
    }
}
```

### Actors

```swift
// Good: Actor for shared mutable state
actor CacheManager {
    private var cache: [String: Data] = [:]
    private let maxSize: Int

    init(maxSize: Int = 100) {
        self.maxSize = maxSize
    }

    func get(_ key: String) -> Data? {
        cache[key]
    }

    func set(_ key: String, data: Data) {
        if cache.count >= maxSize {
            evictOldest()
        }
        cache[key] = data
    }

    private func evictOldest() {
        // Eviction logic
    }
}

// Usage
let cacheManager = CacheManager()

Task {
    if let cached = await cacheManager.get("route-123") {
        return cached
    }
    let data = try await fetchData()
    await cacheManager.set("route-123", data: data)
}
```

### MainActor

```swift
// Good: ViewModel on MainActor
@MainActor
final class RouteViewModel: ObservableObject {
    @Published var routes: [Route] = []

    func loadRoutes() async {
        // Already on MainActor, safe to update @Published
        routes = try await routeService.fetchRoutes()
    }
}

// Good: Explicit MainActor for specific methods
final class DataProcessor {
    func processData(_ data: Data) async throws -> ProcessedData {
        // Heavy processing on background thread
        let processed = try await heavyProcessing(data)

        // Update UI on main thread
        await updateUI(with: processed)

        return processed
    }

    @MainActor
    private func updateUI(with data: ProcessedData) {
        // Safe to update UI
    }
}
```

### Cancellation

```swift
// Good: Check for cancellation
func fetchAllPages() async throws -> [Page] {
    var pages: [Page] = []
    var currentPage = 1

    while true {
        // Check if task was cancelled
        try Task.checkCancellation()

        let page = try await fetchPage(currentPage)
        pages.append(contentsOf: page.items)

        if page.isLastPage { break }
        currentPage += 1
    }

    return pages
}

// Good: Handle cancellation gracefully
func loadData() async {
    do {
        let data = try await fetchData()
        self.data = data
    } catch is CancellationError {
        // Task was cancelled, clean up
        return
    } catch {
        self.error = error
    }
}
```

---

## Memory Management

### Reference Cycles

```swift
// Good: Weak self in closures
class RouteViewModel {
    func startLocationUpdates() {
        locationManager.onUpdate = { [weak self] location in
            self?.handleLocation(location)
        }
    }
}

// Good: Capture list when needed
class ImageLoader {
    func loadImage(url: URL, completion: @escaping (UIImage?) -> Void) {
        URLSession.shared.dataTask(with: url) { [weak self] data, _, _ in
            guard let self = self else { return }
            let image = self.processData(data)
            completion(image)
        }.resume()
    }
}

// Good: Unowned when lifetime is guaranteed
class Parent {
    lazy var child: Child = {
        Child(parent: self)
    }()
}

class Child {
    unowned let parent: Parent

    init(parent: Parent) {
        self.parent = parent
    }
}
```

### Value Types vs Reference Types

```swift
// Prefer structs for data models
struct Route: Identifiable, Hashable {
    let id: UUID
    var segments: [Segment]
    var exposureScore: Double
}

// Use classes for:
// - Identity is important (not just values)
// - Shared mutable state
// - Inheritance is needed
// - Interop with Objective-C

// Good: Class for service with identity
final class RouteService {
    private let apiClient: APIClient

    init(apiClient: APIClient) {
        self.apiClient = apiClient
    }
}
```

### Deinitialization

```swift
// Good: Clean up resources
final class LocationTracker {
    private var timer: Timer?
    private var locationManager: CLLocationManager?

    deinit {
        timer?.invalidate()
        locationManager?.stopUpdatingLocation()
    }
}

// Good: Remove observers
final class NotificationObserver {
    private var observation: NSObjectProtocol?

    init() {
        observation = NotificationCenter.default.addObserver(
            forName: .userDidLogout,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            self?.handleLogout()
        }
    }

    deinit {
        if let observation = observation {
            NotificationCenter.default.removeObserver(observation)
        }
    }
}
```

---

## Testing

### Unit Tests

```swift
import XCTest
@testable import BreatheRoute

final class RouteViewModelTests: XCTestCase {

    // MARK: - Properties

    private var sut: RouteViewModel!
    private var mockRouteService: MockRouteService!

    // MARK: - Setup

    override func setUp() {
        super.setUp()
        mockRouteService = MockRouteService()
        sut = RouteViewModel(routeService: mockRouteService)
    }

    override func tearDown() {
        sut = nil
        mockRouteService = nil
        super.tearDown()
    }

    // MARK: - Tests

    func test_loadRoutes_whenSuccess_updatesRoutes() async {
        // Given
        let expectedRoutes = [Route.mock(), Route.mock()]
        mockRouteService.routesToReturn = expectedRoutes

        // When
        await sut.loadRoutes()

        // Then
        XCTAssertEqual(sut.routes, expectedRoutes)
        XCTAssertNil(sut.error)
        XCTAssertFalse(sut.isLoading)
    }

    func test_loadRoutes_whenFailure_setsError() async {
        // Given
        mockRouteService.errorToThrow = NetworkError.noConnection

        // When
        await sut.loadRoutes()

        // Then
        XCTAssertTrue(sut.routes.isEmpty)
        XCTAssertNotNil(sut.error)
    }

    func test_calculateExposure_withValidRoute_returnsScore() {
        // Given
        let route = Route.mock(segments: [
            .mock(pollution: 20),
            .mock(pollution: 40),
            .mock(pollution: 30)
        ])

        // When
        let score = sut.calculateExposure(for: route)

        // Then
        XCTAssertEqual(score, 30, accuracy: 0.01)
    }
}
```

### Test Naming

```swift
// Pattern: test_methodName_condition_expectedResult
func test_fetchCommutes_whenAuthenticated_returnsCommutes() async
func test_fetchCommutes_whenUnauthorized_throwsError() async
func test_deleteCommute_whenNotFound_throwsNotFoundError() async
func test_validateEmail_withInvalidFormat_returnsFalse()
```

### Mocking

```swift
// Mock with configurable behavior
final class MockRouteService: RouteServiceProtocol {
    // Track method calls
    var fetchRoutesCalled = false
    var fetchRoutesCallCount = 0
    var lastFetchRoutesOrigin: Coordinate?

    // Configure return values
    var routesToReturn: [Route] = []
    var errorToThrow: Error?

    func fetchRoutes(
        from origin: Coordinate,
        to destination: Coordinate
    ) async throws -> [Route] {
        fetchRoutesCalled = true
        fetchRoutesCallCount += 1
        lastFetchRoutesOrigin = origin

        if let error = errorToThrow {
            throw error
        }

        return routesToReturn
    }
}

// Test helpers
extension Route {
    static func mock(
        id: UUID = UUID(),
        mode: RouteMode = .bike,
        exposureScore: Double = 50.0
    ) -> Route {
        Route(
            id: id,
            mode: mode,
            segments: [],
            duration: 1800,
            distance: 5000,
            exposureScore: exposureScore
        )
    }
}
```

### Async Testing

```swift
func test_fetchData_withTimeout() async throws {
    // Given
    let expectation = XCTestExpectation(description: "Data fetched")

    // When
    Task {
        try await sut.fetchData()
        expectation.fulfill()
    }

    // Then
    await fulfillment(of: [expectation], timeout: 5.0)
}

func test_publishedProperty_updates() async {
    // Given
    let expectedRoutes = [Route.mock()]
    mockService.routesToReturn = expectedRoutes

    // When
    await sut.loadRoutes()

    // Then - wait for @Published to update
    try await Task.sleep(nanoseconds: 100_000_000)
    XCTAssertEqual(sut.routes, expectedRoutes)
}
```

### UI Testing

```swift
import XCTest

final class CommuteListUITests: XCTestCase {

    private var app: XCUIApplication!

    override func setUpWithError() throws {
        continueAfterFailure = false
        app = XCUIApplication()
        app.launchArguments = ["--uitesting"]
        app.launch()
    }

    func test_addCommute_showsNewCommute() {
        // Navigate to commutes
        app.tabBars.buttons["Commutes"].tap()

        // Tap add button
        app.navigationBars.buttons["Add"].tap()

        // Fill form
        app.textFields["Label"].tap()
        app.textFields["Label"].typeText("Work Commute")

        // Save
        app.buttons["Save"].tap()

        // Verify
        XCTAssertTrue(app.cells["Work Commute"].exists)
    }
}
```

---

## Security

### Keychain Storage

```swift
import Security

final class KeychainManager {

    enum KeychainError: Error {
        case duplicateItem
        case itemNotFound
        case unexpectedStatus(OSStatus)
    }

    func save(_ data: Data, forKey key: String) throws {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlock
        ]

        let status = SecItemAdd(query as CFDictionary, nil)

        switch status {
        case errSecSuccess:
            return
        case errSecDuplicateItem:
            try update(data, forKey: key)
        default:
            throw KeychainError.unexpectedStatus(status)
        }
    }

    func retrieve(forKey key: String) throws -> Data {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecReturnData as String: true
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        guard status == errSecSuccess else {
            throw KeychainError.itemNotFound
        }

        guard let data = result as? Data else {
            throw KeychainError.itemNotFound
        }

        return data
    }
}
```

### Secure Networking

```swift
// Use URLSession with proper configuration
final class SecureAPIClient {
    private let session: URLSession

    init() {
        let configuration = URLSessionConfiguration.default
        configuration.tlsMinimumSupportedProtocolVersion = .TLSv12
        configuration.httpAdditionalHeaders = [
            "Content-Type": "application/json",
            "Accept": "application/json"
        ]

        self.session = URLSession(configuration: configuration)
    }

    func request<T: Decodable>(_ endpoint: Endpoint) async throws -> T {
        var request = URLRequest(url: endpoint.url)
        request.httpMethod = endpoint.method.rawValue

        // Add auth header if needed
        if let token = try? KeychainManager.shared.retrieveToken() {
            request.addValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        let (data, response) = try await session.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw NetworkError.invalidResponse
        }

        guard (200...299).contains(httpResponse.statusCode) else {
            throw NetworkError.httpError(statusCode: httpResponse.statusCode)
        }

        return try JSONDecoder().decode(T.self, from: data)
    }
}
```

### Input Validation

```swift
// Validate all user input
struct EmailValidator {
    private static let emailRegex = /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/
        .ignoresCase()

    static func validate(_ email: String) -> Bool {
        email.wholeMatch(of: emailRegex) != nil
    }
}

// Sanitize before display
extension String {
    var sanitizedForDisplay: String {
        self
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .replacingOccurrences(of: "<", with: "&lt;")
            .replacingOccurrences(of: ">", with: "&gt;")
    }
}
```

---

## Performance

### Lazy Loading

```swift
// Good: Lazy initialization
final class HeavyObject {
    lazy var expensiveData: Data = {
        // Only computed when first accessed
        loadExpensiveData()
    }()
}

// Good: Lazy views in SwiftUI
struct LazyLoadingList: View {
    var body: some View {
        ScrollView {
            LazyVStack {
                ForEach(items) { item in
                    ItemRow(item: item)  // Only created when visible
                }
            }
        }
    }
}
```

### Image Optimization

```swift
// Good: AsyncImage with caching
AsyncImage(url: route.thumbnailURL) { phase in
    switch phase {
    case .empty:
        ProgressView()
    case .success(let image):
        image
            .resizable()
            .aspectRatio(contentMode: .fill)
    case .failure:
        Image(systemName: "photo")
    @unknown default:
        EmptyView()
    }
}
.frame(width: 80, height: 80)

// Good: Downsampling large images
extension UIImage {
    static func downsample(
        imageAt url: URL,
        to pointSize: CGSize,
        scale: CGFloat = UIScreen.main.scale
    ) -> UIImage? {
        let imageSourceOptions = [kCGImageSourceShouldCache: false] as CFDictionary
        guard let imageSource = CGImageSourceCreateWithURL(url as CFURL, imageSourceOptions) else {
            return nil
        }

        let maxDimensionInPixels = max(pointSize.width, pointSize.height) * scale
        let downsampleOptions = [
            kCGImageSourceCreateThumbnailFromImageAlways: true,
            kCGImageSourceShouldCacheImmediately: true,
            kCGImageSourceCreateThumbnailWithTransform: true,
            kCGImageSourceThumbnailMaxPixelSize: maxDimensionInPixels
        ] as CFDictionary

        guard let downsampledImage = CGImageSourceCreateThumbnailAtIndex(imageSource, 0, downsampleOptions) else {
            return nil
        }

        return UIImage(cgImage: downsampledImage)
    }
}
```

### Avoid Unnecessary Work

```swift
// Good: Debounce search input
final class SearchViewModel: ObservableObject {
    @Published var searchText = ""
    @Published private(set) var results: [Result] = []

    private var searchTask: Task<Void, Never>?

    init() {
        $searchText
            .debounce(for: .milliseconds(300), scheduler: DispatchQueue.main)
            .removeDuplicates()
            .sink { [weak self] text in
                self?.search(query: text)
            }
            .store(in: &cancellables)
    }

    private func search(query: String) {
        searchTask?.cancel()

        searchTask = Task {
            guard !query.isEmpty else {
                results = []
                return
            }

            do {
                results = try await searchService.search(query: query)
            } catch is CancellationError {
                // Ignore cancellation
            } catch {
                // Handle error
            }
        }
    }
}
```

---

## Accessibility

### VoiceOver Support

```swift
// Good: Meaningful labels
Button {
    viewModel.deleteCommute(commute)
} label: {
    Image(systemName: "trash")
}
.accessibilityLabel("Delete \(commute.label)")
.accessibilityHint("Double tap to delete this commute")

// Good: Group related elements
HStack {
    Image(systemName: route.mode.iconName)
    Text(route.duration.formatted())
    Text(route.distance.formatted())
}
.accessibilityElement(children: .combine)
.accessibilityLabel("\(route.mode.name) route, \(route.duration.formatted()), \(route.distance.formatted())")

// Good: Custom actions
RouteCard(route: route)
    .accessibilityAction(named: "Select as favorite") {
        viewModel.toggleFavorite(route)
    }
    .accessibilityAction(named: "Share route") {
        viewModel.share(route)
    }
```

### Dynamic Type

```swift
// Good: Use system fonts that scale
Text("Route Details")
    .font(.headline)  // Scales automatically

Text("Distance: 5.2 km")
    .font(.body)

// Good: Custom fonts that scale
Text("Custom Text")
    .font(.custom("Avenir", size: 16, relativeTo: .body))

// Good: Handle large text
struct RouteRow: View {
    @Environment(\.dynamicTypeSize) var dynamicTypeSize

    var body: some View {
        if dynamicTypeSize >= .accessibility1 {
            // Stack vertically for large text
            VStack(alignment: .leading) {
                modeIcon
                routeDetails
            }
        } else {
            // Horizontal layout for regular text
            HStack {
                modeIcon
                routeDetails
            }
        }
    }
}
```

### Color and Contrast

```swift
// Good: Sufficient contrast
struct ExposureIndicator: View {
    let score: Double

    private var color: Color {
        switch score {
        case 0..<30: return .green
        case 30..<60: return .orange
        default: return .red
        }
    }

    var body: some View {
        HStack {
            Circle()
                .fill(color)
                .frame(width: 12, height: 12)
            Text(scoreLabel)
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Exposure: \(scoreLabel)")
    }
}

// Good: Don't rely on color alone
struct StatusBadge: View {
    let status: Status

    var body: some View {
        Label(status.title, systemImage: status.iconName)
            .foregroundColor(status.color)
            .accessibilityLabel(status.accessibilityLabel)
    }
}
```

---

## Project Structure

### Folder Organization

```
ios/
├── BreatheRoute/
│   ├── App/
│   │   ├── BreatheRouteApp.swift
│   │   ├── AppDelegate.swift
│   │   └── SceneDelegate.swift
│   ├── Core/
│   │   ├── DependencyContainer.swift
│   │   ├── Configuration.swift
│   │   └── Constants.swift
│   ├── Features/
│   │   ├── Auth/
│   │   │   ├── Views/
│   │   │   ├── ViewModels/
│   │   │   └── Models/
│   │   ├── Routes/
│   │   │   ├── Views/
│   │   │   ├── ViewModels/
│   │   │   └── Models/
│   │   ├── Commutes/
│   │   └── Alerts/
│   ├── Services/
│   │   ├── API/
│   │   ├── Location/
│   │   ├── Push/
│   │   └── Analytics/
│   ├── Repositories/
│   │   ├── CommuteRepository.swift
│   │   └── RouteRepository.swift
│   ├── Utilities/
│   │   ├── Extensions/
│   │   ├── Helpers/
│   │   └── Validators/
│   └── Resources/
│       ├── Assets.xcassets
│       ├── Localizable.strings
│       └── Info.plist
├── BreatheRouteTests/
│   ├── Features/
│   ├── Services/
│   ├── Mocks/
│   └── Helpers/
└── BreatheRouteUITests/
```

### File Organization

```swift
// Order within a file:
// 1. Import statements
// 2. Type declaration
// 3. Nested types
// 4. Properties (static, then instance)
// 5. Initializers
// 6. Lifecycle methods
// 7. Public methods
// 8. Private methods

import SwiftUI
import Combine

final class RouteViewModel: ObservableObject {

    // MARK: - Nested Types

    enum State {
        case idle
        case loading
        case loaded([Route])
        case error(Error)
    }

    // MARK: - Published Properties

    @Published private(set) var state: State = .idle

    // MARK: - Private Properties

    private let routeService: RouteServiceProtocol
    private var cancellables = Set<AnyCancellable>()

    // MARK: - Initialization

    init(routeService: RouteServiceProtocol) {
        self.routeService = routeService
    }

    // MARK: - Public Methods

    func loadRoutes() async {
        state = .loading
        // ...
    }

    // MARK: - Private Methods

    private func processRoutes(_ routes: [Route]) -> [Route] {
        // ...
    }
}
```

---

## Dependencies

### Swift Package Manager

```swift
// Package.swift or via Xcode
dependencies: [
    .package(url: "https://github.com/pointfreeco/swift-composable-architecture", from: "1.0.0"),
    .package(url: "https://github.com/Alamofire/Alamofire", from: "5.8.0"),
]
```

### Recommended Libraries

| Purpose | Library |
|---------|---------|
| Networking | URLSession (built-in) or Alamofire |
| Image Loading | AsyncImage (built-in) or Kingfisher |
| Keychain | KeychainAccess |
| Analytics | Firebase Analytics |
| Crash Reporting | Firebase Crashlytics |
| Linting | SwiftLint |
| Testing | Quick/Nimble (optional) |

### Dependency Guidelines

1. **Prefer built-in solutions** - URLSession, Codable, async/await
2. **Minimize dependencies** - Each adds maintenance burden
3. **Evaluate carefully** - Check maintenance, community, license
4. **Pin versions** - Use exact versions for stability
5. **Document usage** - Explain why each dependency is needed

---

## Code Review Checklist

### Correctness
- [ ] Does the code work as intended?
- [ ] Are edge cases handled?
- [ ] Are optionals safely unwrapped?
- [ ] Is error handling complete?

### Architecture
- [ ] Does code follow MVVM pattern?
- [ ] Are dependencies injected?
- [ ] Is business logic in ViewModels, not Views?
- [ ] Are protocols used for testability?

### SwiftUI
- [ ] Are views small and focused?
- [ ] Is state management appropriate?
- [ ] Are expensive computations avoided in body?
- [ ] Is navigation handled correctly?

### Concurrency
- [ ] Is @MainActor used for UI updates?
- [ ] Are Tasks properly cancelled?
- [ ] Are race conditions avoided?
- [ ] Is async/await used idiomatically?

### Memory
- [ ] Are retain cycles avoided?
- [ ] Is [weak self] used in closures?
- [ ] Are resources properly cleaned up?

### Testing
- [ ] Are unit tests added for new code?
- [ ] Do tests cover edge cases?
- [ ] Are mocks used appropriately?

### Accessibility
- [ ] Are accessibility labels meaningful?
- [ ] Does the UI work with Dynamic Type?
- [ ] Is color contrast sufficient?

### Performance
- [ ] Are images optimized?
- [ ] Is lazy loading used where appropriate?
- [ ] Are expensive operations off the main thread?

### Style
- [ ] Does code pass SwiftLint?
- [ ] Are naming conventions followed?
- [ ] Is code properly documented?
- [ ] Is MARK used for organization?

---

## References

- [Swift API Design Guidelines](https://swift.org/documentation/api-design-guidelines/)
- [Apple Human Interface Guidelines](https://developer.apple.com/design/human-interface-guidelines/)
- [SwiftUI Documentation](https://developer.apple.com/documentation/swiftui/)
- [Swift Concurrency](https://docs.swift.org/swift-book/LanguageGuide/Concurrency.html)
- [Ray Wenderlich Swift Style Guide](https://github.com/raywenderlich/swift-style-guide)
- [Google Swift Style Guide](https://google.github.io/swift/)
