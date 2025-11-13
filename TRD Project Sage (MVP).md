# **Technical Requirements Document (TRD)**

# **Project Sage (MVP)**

### **1\. Executive Summary**

Project Sage is a mobile-first application designed to serve the elderly community. It has two primary functions:

1. **Assistance:** To provide on-demand, chat-based help for "life skills" (initially focused on concrete tasks like IT, home, and auto questions) via a hybrid LLM and human-expert model.  
2. **Social:** To provide an engaging, LLM-powered social chat experience to combat loneliness.

This document outlines the technical requirements for the Minimum Viable Product (MVP). The MVP's core objective is to validate the LLM-to-human handoff, the utility of the assistance model, and the usability of the app for the target demographic.

### **2\. Project Scope**

#### **2.1. In-Scope (MVP)**

* **User Application (Flutter):**  
  * User onboarding & authentication (via Firebase).  
  * Simple user profile management.  
  * LLM-powered "Social Chat" (Gemini).  
  * "Request Expert Help" workflow.  
  * Chat interface (Twilio) for both LLM and Human Expert conversations.  
  * Ability to send/receive text, photos, and video *files* (no live stream).  
  * Membership tier visibility (e.g., "5 Assistance Tokens remaining").  
  * Ability to rate an expert after a session.  
* **Expert Application (Flutter):**  
  * A separate app for the internal support team.  
  * Expert authentication.  
  * A queue of incoming assistance requests.  
  * Ability to accept a request (which removes it from the queue).  
  * View of the LLM-generated summary and the full user-LLM chat history.  
  * Chat interface (Twilio) to communicate with the user.  
  * Ability to send a "Time Estimate" (e.g., "This will use 1 token / \~20 mins") to the user for approval.  
  * Ability to mark a request as "Resolved."  
* **Backend (Golang Microservices):**  
  * User & Expert authentication service (integrating with Firebase).  
  * Profile management service.  
  * A "Request Service" to manage the queue and lifecycle of assistance requests.  
  * An "LLM Gateway Service" to manage all interactions with the Gemini API (for both social chat and summarization).  
  * A "Chat Gateway Service" to manage Twilio Conversations sessions and webhooks.  
  * A "Billing Service" to manage user tokens.  
* **Infrastructure:**  
  * **Postgres (Cloud SQL):** Primary operational database.  
  * **GCS:** Media storage.  
  * **Firebase:** Authentication.  
  * **Twilio:** Conversations API for all chat functionality.  
  * **BigQuery:** Analytics (via ETL pipeline from Postgres/Twilio).

#### **2.2. Out-of-Scope**

* Live video/voice chat.  
* Crowdsourcing model for external "experts."  
* Expert certifications, automated routing, and public ratings.  
* High-liability assistance categories (medical, financial, mental health).  
* Advanced social features (e.g., group chats, forums).  
* Direct user-to-user payments.

#### **2.3. Assumptions**

* The target audience (elderly) will have access to a modern iOS or Android smartphone.  
* The initial "experts" are a small, trained internal support team.  
* Twilio Conversations API will meet the MVP's chat, media, and handoff requirements.  
* Users will find value in a hybrid LLM/human support model.

### **3\. User Roles & Personas**

1. **Standard User**  
   * **Goal:** Wants to feel connected and have a trusted, simple-to-use source of help for technology, home, or other daily tasks.  
   * **Needs:** High accessibility (large fonts, simple UI, clear buttons), patience, and trust. May be skeptical of technology.  
   * **Key Action:** Starting a social chat, escalating to human help.  
2. **Expert User (Internal Support Staff)**  
   * **Goal:** To efficiently and accurately resolve a user's assistance request.  
   * **Needs:** A clear, prioritized queue; full context of the user's problem; simple tools to communicate and close requests.  
   * **Key Action:** Accepting a request, reviewing history, and providing a solution.

### **4\. System Architecture**

The system will be a service-oriented architecture (SOA) using Golang for backend services, Flutter for mobile apps, and managed Google Cloud/Firebase/Twilio services.

#### **4.1. High-Level Diagram**

| \[Flutter User App\] \----\> \[API Gateway\] \----\> \[UserService (Go)\]     |                      |                     |     |                      |                     \+-\> \[RequestService (Go)\]     |                      |                     |     |                      |                     \+-\> \[BillingService (Go)\]     |                      |                     |     v                      v                     v\[Twilio Conversations\] \<-\> \[ChatGateway (Go)\]   \[Postgres DB (Cloud SQL)\]     |                      |                     ^     |                      |                     |\[Flutter Expert App\] \---\> \[API Gateway\]           |\---\[GCS (Media)\]     |                                            |     v                                            |\[Gemini API\] \<--------- \[LLM Gateway (Go)\] \-------+ |
| :---- |

#### **4.2. Backend Services (Golang)**

* **UserService:** Manages user/expert profiles, preferences, and auth state. (Data in Postgres).  
* **RequestService:** Manages the assistance\_request lifecycle (pending, active, resolved). Handles queue logic. (Data in Postgres).  
* **BillingService:** Manages user token balances. POST /token/debit (called by RequestService). (Data in Postgres).  
* **LLMGatewayService:**  
  * Provides a unified interface for all Gemini calls.  
  * POST /chat/social: For general LLM chat.  
  * POST /chat/summarize: (Internal) Called by RequestService on handoff.  
* **ChatGatewayService:**  
  * Manages Twilio Conversation creation, adding participants (user, LLM, expert), and handling media.  
  * Generates Twilio auth tokens for the client apps.  
  * Acts as the proxy for all chat messages.

### **5\. Functional Requirements (User Stories)**

#### **5.1. Module 1: Onboarding & Core App**

* **U-1.1 (User):** As a new user, I can sign up using my email/password or phone number (Firebase Auth) so I can create an account.  
* **U-1.2 (User):** As a user, I can create a simple profile with my name and an optional photo.  
* **U-1.3 (User):** As a user, I can clearly see my membership status and how many "Assistance Tokens" I have.  
* **U-1.4 (User):** As a user, the app interface must be simple, with large, easy-to-read text (e.g., Inter font) and high-contrast buttons (Accessibility NFR).

#### **5.2. Module 2: Social Chat (LLM)**

* **U-2.1 (User):** As a user, I can open the app and immediately start a "social chat" with the AI assistant (Gemini).  
* **U-2.2 (User):** As a user, I can send text messages, photos from my camera/gallery, and short video files to the AI.  
* **U-2.3 (User):** As a user, I expect the AI's responses to be conversational and supportive.

#### **5.3. Module 3: Assistance Handoff**

* **U-3.1 (User):** As a user, I can see a prominent "Request Expert Help" button at all times within the chat interface.  
* **U-3.2 (User):** As a user, when I tap "Request Expert Help," I am shown a confirmation modal that says, "This will use 1 Assistance Token. Connect to a human expert?"  
* **U-3.3 (User):** As a user, after confirming, I am placed in a queue and see a message like, "You're in line\! An expert will join this chat shortly."  
* **U-3.4 (Backend):** As the system, when a user confirms a request, I will:  
  1. Call the BillingService to debit one token.  
  2. Call the LLMGatewayService to summarize the recent chat history.  
  3. Create a new assistance\_request record in Postgres with status pending and the llm\_summary.

#### **5.4. Module 4: Expert Chat**

* **U-4.1 (User):** As a user, I am notified in the *same chat window* when an expert joins (e.g., "Joe has joined the chat").  
* **U-4.2 (User):** As a user, I *do not* have to repeat my problem, as the expert has my chat history.  
* **U-4.3 (User):** As a user, I can receive a "Time Estimate" from the expert (e.g., "This looks like a 20-minute task / 1 token"). I must be able to approve this before the expert proceeds.  
* **U-4.4 (User):** As a user, after the request is marked "Resolved," I am prompted to leave a simple 1-5 star rating for the expert.  
* **U-4.5 (Expert):** As an expert, I can log in to the Expert App.  
* **U-4.6 (Expert):** As an expert, I can see a list of all pending assistance requests, sorted by wait time.  
* **U-4.7 (Expert):** As an expert, I can tap "Accept" on a request. This updates its status to active and assigns me.  
* **U-4.8 (Expert):** As an expert, when I accept, I see a modal with the llm\_summary.  
* **U-4.9 (Expert):** As an expert, I can scroll up in the chat window to read the user's full history with the LLM.  
* **U-4.10 (Expert):** As an expert, I have a button to "Send Time Estimate" to the user.  
* **U-4.11 (Expert):** As an expert, I can mark the chat "Resolved," which closes the request and triggers the rating prompt for the user.

### **6\. Non-Functional Requirements (NFRs)**

* **Accessibility (CRITICAL):**  
  * Must meet WCAG 2.1 AA guidelines.  
  * Default font size must be large (e.g., 18pt), and all UI must support dynamic text scaling.  
  * All interactive elements must have a tap-target size of at least 48x48dp.  
  * Must include a high-contrast mode.  
* **Security:**  
  * All PII (user/expert data) must be encrypted at rest (Postgres) and in transit (SSL).  
  * Media stored in GCS must be private. Access will be via short-lived, pre-signed URLs generated by the backend.  
  * Strict IAM roles for all Golang services.  
  * Prominent disclaimers regarding advice ("This is not professional, certified advice...") must be shown and agreed to by the user.  
* **Performance:**  
  * App cold start \< 4 seconds.  
  * Chat messages (Twilio) \< 1.5s delivery.  
  * LLM social chat responses \< 3s (p90).  
  * Expert queue must update in near-real-time (\< 10s poll).  
* **Scalability:**  
  * All Golang services must be containerized (e.g., for Cloud Run or GKE) and stateless.  
  * Chat scaling is offloaded to Twilio.  
  * Database (Cloud SQL) will use a primary instance with read-replicas.

### **7\. Technology Stack**

* **Client (User & Expert):** Flutter  
* **Backend Services:** Golang  
* **Primary Database:** Postgres (Google Cloud SQL)  
* **Analytics Database:** Google BigQuery  
* **Authentication:** Firebase Authentication  
* **Chat Infrastructure:** Twilio Conversations API  
* **Media Storage:** Google Cloud Storage (GCS)  
* **AI/LLM:** Google Gemini API  
* **Hosting/Infra:** Google Cloud Platform (GKE or Cloud Run)

### **8\. Data Model & Storage**

#### **8.1. Postgres (Cloud SQL)**

* **users**:  
  * user\_id (UUID, PK)  
  * firebase\_auth\_id (String)  
  * display\_name (String)  
  * profile\_image\_url (String)  
  * membership\_tier (String, e.g., 'free', 'premium')  
  * assistance\_token\_balance (Int)  
* **experts**:  
  * expert\_id (UUID, PK)  
  * firebase\_auth\_id (String)  
  * display\_name (String, e.g., "Joe from Support")  
  * is\_active (Bool)  
* **assistance\_requests**:  
  * request\_id (UUID, PK)  
  * user\_id (FK, users)  
  * expert\_id (FK, experts, nullable)  
  * status (Enum: 'pending', 'active', 'resolved', 'cancelled')  
  * llm\_summary (Text)  
  * twilio\_conversation\_sid (String)  
  * created\_at (Timestamp)  
  * accepted\_at (Timestamp)  
  * resolved\_at (Timestamp)  
* **expert\_ratings**:  
  * rating\_id (UUID, PK)  
  * request\_id (FK, assistance\_requests)  
  * user\_id (FK, users)  
  * expert\_id (FK, experts)  
  * score (Int, 1-5)

#### **8.2. GCS**

* One bucket (project-sage-media-prod) with folders for user\_media and expert\_media.  
* Files named with a UUID (e.g., user\_media/abc-123/my\_photo.jpg).  
* Media is private by default.

#### **8.3. BigQuery**

* Data loaded via ETL from Postgres (e.g., daily batch).  
* Datasets: app\_analytics, request\_metrics.  
* Tables: anonymized\_requests (for analysis), user\_activity.

### **9\. LLM-Human Handoff**

1. **User (User App)** chats with **Gemini (LLMGateway)**. The chat occurs within a Twilio Conversation (twilio\_conversation\_sid) where the "LLM Bot" is a participant.  
2. **User** taps "Request Expert Help."  
3. **User App** shows a confirmation modal. User taps "Confirm."  
4. **User App** \-\> POST /api/v1/request/create (Body: { twilio\_sid: '...' }).  
5. RequestService receives the call.  
   a. Calls BillingService \-\> POST /api/v1/token/debit (Body: { user\_id: '...' }).  
   b. If successful, RequestService calls LLMGateway \-\> POST /api/v1/chat/summarize (Body: { twilio\_sid: '...' }).  
   c. RequestService creates a new assistance\_requests record in Postgres with status: 'pending' and the llm\_summary.  
   d. RequestService removes the "LLM Bot" participant from the Twilio Conversation (to prevent it from talking).  
6. **Expert App** is polling GET /api/v1/request/pending every 10 seconds. The new request appears in the queue.  
7. **Expert** taps "Accept."  
8. **Expert App** \-\> POST /api/v1/request/accept (Body: { request\_id: '...' }).  
9. RequestService receives the call.  
   a. Updates assistance\_requests record: status: 'active', expert\_id: '...'.  
   b. Calls ChatGateway (Twilio) to add the expert as a participant to the twilio\_conversation\_sid.  
10. **User App** receives a Twilio event: "Participant 'Expert Joe' has joined." The UI updates.  
11. The chat is now live between the **User** and the **Expert** in the same window.