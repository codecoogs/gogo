# GET Routes Documentation

This document describes all GET endpoints in the GoGo API and how to use them.  
Base URL: `https://api.codecoogs.com/v1` (or your deployment URL). Routes are also available under `/api/` without the `/v1` prefix if not using the rewrite.

---

## 1. Users – `/v1/users`

### 1.1 Get all users with payment/due date info

**Use case:** List every user with last payment date, next due date, and paid status (e.g. admin dashboard).

| Item             | Value                              |
| ---------------- | ---------------------------------- |
| **URL**          | `GET /v1/users?payment_info=true`  |
| **Query params** | `payment_info` = `true` (required) |
| **Auth**         | None                               |

**Example request:**

```http
GET /v1/users?payment_info=true
```

**Example response:**

```json
{
  "success": true,
  "users_payment_info": [
    {
      "id": "uuid",
      "first_name": "Jane",
      "last_name": "Doe",
      "email": "jane@example.com",
      "phone": "...",
      "major": "...",
      "classification": "...",
      "expected_graduation": "...",
      "membership": "Yearly",
      "discord": "...",
      "paid": true,
      "last_payment_date": "2024-01-15T00:00:00Z",
      "next_due_date": "2025-01-15T00:00:00Z"
    }
  ]
}
```

- **`next_due_date`** is set only for users with membership `Yearly` or `Semester` and a known last payment (Yearly = +1 year, Semester = +6 months from last payment). Otherwise it is omitted/empty.
- **`last_payment_date`** comes from the most recent payment record, or from the user’s `updated` timestamp when `paid` is true and no payment row exists.

---

### 1.2 Get active members only

**Use case:** List members whose membership is still active (due date in the future).

| Item             | Value                                    |
| ---------------- | ---------------------------------------- |
| **URL**          | `GET /v1/users?active_memberships=true`  |
| **Query params** | `active_memberships` = `true` (required) |
| **Auth**         | None                                     |

**Example request:**

```http
GET /v1/users?active_memberships=true
```

**Example response:**

```json
{
  "success": true,
  "active_members": [
    {
      "id": "uuid",
      "first_name": "Jane",
      "last_name": "Doe",
      "email": "jane@example.com",
      "phone": "...",
      "major": "...",
      "classification": "...",
      "expected_graduation": "...",
      "membership": "Yearly",
      "discord": "...",
      "due_date": "2025-01-15T00:00:00Z",
      "last_payment_date": "2024-01-15T00:00:00Z"
    }
  ]
}
```

- Only users with membership **Yearly** or **Semester** and a **due date after now** are returned.

---

### 1.3 Get a single user by ID

**Use case:** Fetch one user’s full record.

| Item             | Value                       |
| ---------------- | --------------------------- |
| **URL**          | `GET /v1/users?id={uuid}`   |
| **Query params** | `id` = user UUID (required) |
| **Auth**         | None                        |

**Example request:**

```http
GET /v1/users?id=550e8400-e29b-41d4-a716-446655440000
```

**Example response:**

```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "first_name": "Jane",
      "last_name": "Doe",
      "email": "jane@example.com",
      "phone": "...",
      "auth_id": "...",
      "major": "...",
      "classification": "...",
      "expected_graduation": "...",
      "membership": "Yearly",
      "paid": true,
      "shirt-bought": false,
      "created": "...",
      "updated": "...",
      "discord": "..."
    }
  ]
}
```

---

## 2. User points – `/v1/users/points`

All of these use **GET** on the same path; behavior is determined by query parameters.

### 2.1 Get point categories

**Use case:** Show users how many points each category is worth (e.g. “Meeting = 5 pts”, “Event = 10 pts”).

| Item             | Value                                  |
| ---------------- | -------------------------------------- |
| **URL**          | `GET /v1/users/points?categories=true` |
| **Query params** | `categories` = `true` (required)       |
| **Auth**         | None                                   |

**Example request:**

```http
GET /v1/users/points?categories=true
```

**Example response:**

```json
{
  "success": true,
  "point_categories": [
    {
      "id": "uuid",
      "name": "General Meeting",
      "points_value": 5,
      "description": "Attend a general body meeting"
    },
    {
      "id": "uuid",
      "name": "Workshop",
      "points_value": 10,
      "description": "Attend a workshop event"
    }
  ]
}
```

- Use this to build UI that explains point values per category (e.g. before/after events).

---

### 2.2 Get a user’s point transactions

**Use case:** Show a user their point history (earned per transaction).

| Item             | Value                                                                         |
| ---------------- | ----------------------------------------------------------------------------- | ----- | --------------- |
| **URL**          | `GET /v1/users/points?transactions=true&{id                                   | email | discordId}=...` |
| **Query params** | `transactions` = `true` **and** exactly one of: `id`, `email`, or `discordId` |
| **Auth**         | None                                                                          |

**Example requests:**

```http
GET /v1/users/points?transactions=true&id=550e8400-e29b-41d4-a716-446655440000
GET /v1/users/points?transactions=true&email=jane@example.com
GET /v1/users/points?transactions=true&discordId=123456789
```

**Example response:**

```json
{
  "success": true,
  "point_transactions": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "category_id": "uuid",
      "event_id": 42,
      "points_earned": 5,
      "created_at": "2024-09-15T14:30:00Z",
      "created_by": "uuid",
      "academic_year_id": "uuid"
    }
  ]
}
```

- **`category_id`** can be matched with **point categories** (from 2.1) to show category name and `points_value`.
- If the user is not found, you get `success: false` and an error message.

---

### 2.3 Get a user’s name and total points

**Use case:** Display a user’s display name and current total points (e.g. profile or leaderboard).

| Item             | Value                                                                             |
| ---------------- | --------------------------------------------------------------------------------- |
| **URL**          | `GET /v1/users/points?id=...` or `&email=...` or `&discordId=...`                 |
| **Query params** | Exactly one of: `id`, `email`, or `discordId` (no `categories` or `transactions`) |
| **Auth**         | None                                                                              |

**Example requests:**

```http
GET /v1/users/points?id=550e8400-e29b-41d4-a716-446655440000
GET /v1/users/points?email=jane@example.com
GET /v1/users/points?discordId=123456789
```

**Example response:**

```json
{
  "success": true,
  "data": {
    "first_name": "Jane",
    "last_name": "Doe",
    "points": 150
  }
}
```

- **Points** come from the `points` table (current/latest total for that user).

---

## Quick reference

| Goal                            | Method | URL                                                              |
| ------------------------------- | ------ | ---------------------------------------------------------------- |
| All users + payment/due info    | GET    | `/v1/users?payment_info=true`                                    |
| Active members only             | GET    | `/v1/users?active_memberships=true`                              |
| Single user by id               | GET    | `/v1/users?id={uuid}`                                            |
| Point categories (point values) | GET    | `/v1/users/points?categories=true`                               |
| User’s point transactions       | GET    | `/v1/users/points?transactions=true&id=...` (or email/discordId) |
| User’s name + total points      | GET    | `/v1/users/points?id=...` (or email/discordId)                   |

---

## Using the routes in your app

1. **Point categories (what’s each category worth)**  
   Call `GET /v1/users/points?categories=true` once (e.g. on app load or settings) and cache the result. Use it to show labels like “General Meeting: 5 pts” or “Workshop: 10 pts.”

2. **A user’s point history**  
   After the user is identified (e.g. by id, email, or discordId), call `GET /v1/users/points?transactions=true&id=...`. Optionally join each transaction’s `category_id` to the categories from (1) to show category name and points per row.

3. **Current total and display name**  
   Call `GET /v1/users/points?id=...` (or email/discordId) to get `first_name`, `last_name`, and `points` for profile or leaderboard.

4. **Membership and due dates**

   - `GET /v1/users?payment_info=true` – all users with last payment and next due (admin/reporting).
   - `GET /v1/users?active_memberships=true` – only members whose membership is still active (due date in the future).

5. **Single user record**  
   Use `GET /v1/users?id={uuid}` when you have the user’s id and need their full user row.

All requests in this doc are GET; no request body. Send `Content-Type: application/json` and optional `Authorization` header if you add auth later.
