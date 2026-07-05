## ADDED Requirements

### Requirement: Mini Program Project Shell
The system SHALL provide a native WeChat Mini Program project shell under the source engineering root so future product features can be implemented in a consistent location.

#### Scenario: Project shell exists
- **WHEN** a developer opens the source engineering root
- **THEN** the WeChat Mini Program source shell is available under `apps/miniprogram`

#### Scenario: Core app configuration exists
- **WHEN** a developer inspects the Mini Program source shell
- **THEN** app-level configuration, app entry, global styles, project configuration, and sitemap configuration are present

### Requirement: Primary Tab Navigation
The system SHALL define five primary Mini Program tab pages matching the MVP product navigation.

#### Scenario: Tab pages are configured
- **WHEN** the Mini Program app configuration is loaded
- **THEN** the tab navigation includes feed, index, AI assistant, sectors, and subscribe pages

#### Scenario: Tab pages have page files
- **WHEN** a developer inspects each primary tab page directory
- **THEN** each page has WXML, WXSS, TypeScript, and page JSON files

### Requirement: Engineering Directory Boundaries
The system SHALL separate page UI, reusable components, domain models, mock data, service access, shared state, utilities, constants, styles, and assets into dedicated directories.

#### Scenario: Shared source areas exist
- **WHEN** a developer inspects the Mini Program source shell
- **THEN** dedicated directories exist for components, models, data, services, store, utils, constants, styles, and assets

#### Scenario: Prototype remains reference-only
- **WHEN** the Mini Program source shell is created
- **THEN** prototype files remain outside the source shell and are not modified by this change

### Requirement: Mock-First Data Boundary
The system SHALL support mock-first development through domain data modules and service wrappers that can later be replaced by real API calls.

#### Scenario: Service boundary exists
- **WHEN** a page needs event, market, sector, AI, report, or subscription data
- **THEN** it can depend on a service module instead of reading prototype files or hard-coding browser DOM state

#### Scenario: Mock data is isolated
- **WHEN** mock content is added for MVP pages
- **THEN** it is placed in dedicated data modules rather than embedded directly in page markup

### Requirement: Frontend Safety Boundary
The system SHALL keep backend secrets, model credentials, payment credentials, direct database access, RAG logic, and Agent orchestration out of the Mini Program source.

#### Scenario: Sensitive backend capability is needed
- **WHEN** the Mini Program needs AI analysis, report generation, payment, subscriptions, or event intelligence
- **THEN** it calls a service/API boundary rather than embedding credentials or backend execution logic in the client

#### Scenario: Analysis content is displayed
- **WHEN** a page displays AI or market analysis content
- **THEN** the UI includes or preserves safety positioning that the content is decision-support information and not direct investment advice
