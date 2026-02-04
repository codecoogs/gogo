# Migration Report: Removing `teams` and `points` from Users Table

## Overview
This report details all changes required in the `/api/users` folder when removing the `team` and `points` columns from the `users` table in Supabase.

---

## Files Requiring Changes

### 1. `/api/users/index.go` ⚠️ **CRITICAL - Multiple Changes Required**

#### **Struct Definitions (Lines 41-62, 64-79, 87-102)**

**Current State:**
- `User` struct (lines 41-62): Contains `Team *uuid.UUID` and `Points int`
- `ActiveMember` struct (lines 64-79): Contains `Team *uuid.UUID` and `Points int`
- `UserQuery` struct (lines 87-102): Contains `Team *uuid.UUID` and `Points int`

**Required Changes:**
1. **Remove fields from structs:**
   - Line 60: Remove `Team *uuid.UUID \`json:"team"\``
   - Line 61: Remove `Points int \`json:"points"\``
   - Line 75: Remove `Team *uuid.UUID \`json:"team"\``
   - Line 76: Remove `Points int \`json:"points"\``
   - Line 99: Remove `Team *uuid.UUID \`json:"team"\``
   - Line 100: Remove `Points int \`json:"points"\``

#### **SQL Query (Line 372)**

**Current State:**
```go
Select("id, first_name, last_name, email, phone, major, classification, expected_graduation, membership, paid, discord, team, points, updated", "exact", false)
```

**Required Change:**
- Remove `team, points` from the SELECT statement
- New query: `Select("id, first_name, last_name, email, phone, major, classification, expected_graduation, membership, paid, discord, updated", "exact", false)`

#### **ActiveMember Assignment (Lines 459-460)**

**Current State:**
```go
Team:               user.Team,
Points:             user.Points,
```

**Required Changes:**
- **Option A (Remove completely):** Delete lines 459-460
- **Option B (Join from new tables):** Replace with JOIN queries to fetch from separate `user_teams` and `user_points` tables

**Recommendation:** Remove these lines initially, then add JOIN logic if you need this data in the response.

#### **User Insert/Update Operations (Lines 187-199, 224)**

**Current State:**
- Line 187: Uses `Select("*", ...)` which will fail if `team` and `points` are removed
- Line 199: `Insert(user, ...)` will fail if struct contains removed fields
- Line 224: `Update(existingUser, ...)` will fail if struct contains removed fields

**Required Changes:**
1. **Line 187:** Change `Select("*", ...)` to explicit column list excluding `team` and `points`
2. **Line 199:** Ensure `User` struct doesn't include `Team` and `Points` before insert
3. **Line 224:** Ensure `User` struct doesn't include `Team` and `Points` before update

#### **GET by ID (Line 299)**

**Current State:**
```go
Select("*", "exact", false)
```

**Required Change:**
- Change to explicit column list: `Select("id, first_name, last_name, email, phone, password, major, classification, expected_graduation, membership, paid, shirt-bought, created, updated, discord", "exact", false)`

#### **PUT Update (Line 326)**

**Current State:**
```go
Update(updatedUser, "", "exact")
```

**Required Change:**
- Ensure `updatedUser` struct doesn't contain `Team` and `Points` fields, or filter them out before update

---

### 2. `/api/users/points/index.go` ⚠️ **MAJOR REFACTOR REQUIRED**

#### **Purpose of File:**
This entire file is dedicated to managing user points. It will need significant refactoring to work with a separate `user_points` table.

#### **Current Implementation:**
- **Line 145:** Queries `users` table: `Select("first_name, last_name, points", ...)`
- **Line 156:** Updates `users` table: `Update(userPoints, ...)`

#### **Required Changes:**

1. **Create new table constant** (if not exists):
   - Add `USER_POINTS_TABLE = "user_points"` to `constants/table_names.go`

2. **Update `getNameAndPointsByColumn` function (Line 143-153):**
   - Change from querying `USER_TABLE` to `USER_POINTS_TABLE`
   - Need to JOIN with `users` table to get `first_name` and `last_name`
   - Query should be: `Select("users.first_name, users.last_name, user_points.points", ...).Join("users", "user_points.user_id", "users.id")`

3. **Update `updateUserPoints` function (Line 155-161):**
   - Change from updating `USER_TABLE` to `USER_POINTS_TABLE`
   - Update query should target: `From(USER_POINTS_TABLE).Update(...).Eq("user_id", value)`

4. **Consider renaming:**
   - File could be renamed to better reflect it's managing a separate table
   - Or keep name but update all logic

---

### 3. `/api/users/discord/verify/index.go` ✅ **NO CHANGES NEEDED**

**Analysis:**
- This file only queries `id` and `discord` fields
- No references to `team` or `points`
- **Status:** Safe, no changes required

---

### 4. `/api/users/roles/index.go` ✅ **NO CHANGES NEEDED**

**Analysis:**
- This file only validates user existence and manages user roles
- No references to `team` or `points`
- **Status:** Safe, no changes required

---

### 5. `/api/users/roles/members/index.go` ✅ **NO CHANGES NEEDED**

**Analysis:**
- This file only grants membership roles
- No references to `team` or `points`
- **Status:** Safe, no changes required

---

## Additional Considerations

### **Other Files Outside `/api/users` That May Be Affected:**

1. **`/api/teams/index.go`** (Referenced in grep results)
   - May need to check if it references `users.team` column
   - Likely already uses separate `teams` table

2. **`/api/payments/stripe/index.go`** (Line 33-34)
   - Contains `Team` and `Points` in a User struct
   - May need updates if this struct is used for user operations

3. **`/api/events/index.go`** (Line 20)
   - Contains `Points` field
   - Verify if this is related to user points

---

## Migration Strategy Recommendations

### **Phase 1: Database Setup**
1. Create new tables:
   - `user_points` table with columns: `id`, `user_id` (FK to users), `points`, `created_at`, `updated_at`
   - `user_teams` table with columns: `id`, `user_id` (FK to users), `team_id` (FK to teams), `created_at`, `updated_at`

2. Migrate existing data:
   - Write SQL migration to copy `users.points` → `user_points.points`
   - Write SQL migration to copy `users.team` → `user_teams.team_id`

### **Phase 2: Code Updates**
1. Update struct definitions (remove fields)
2. Update SQL queries (remove columns from SELECT)
3. Refactor `/api/users/points/index.go` to use new table
4. Update any JOIN queries if team/points data is needed in responses

### **Phase 3: Testing**
1. Test all endpoints in `/api/users` folder
2. Verify points operations work with new table
3. Verify team assignments work with new table
4. Test active memberships endpoint (should still work if team/points removed from response)

### **Phase 4: Database Cleanup**
1. Remove `team` and `points` columns from `users` table
2. Verify all functionality still works

---

## Summary of Required Changes

| File | Changes Required | Priority | Complexity |
|------|-----------------|----------|------------|
| `/api/users/index.go` | 8+ changes | **CRITICAL** | High |
| `/api/users/points/index.go` | Complete refactor | **CRITICAL** | High |
| `/api/users/discord/verify/index.go` | None | Low | N/A |
| `/api/users/roles/index.go` | None | Low | N/A |
| `/api/users/roles/members/index.go` | None | Low | N/A |

---

## Risk Assessment

**High Risk Areas:**
1. Active memberships endpoint may break if `team`/`points` are in response but not in query
2. User creation/update operations will fail if structs contain removed fields
3. Points endpoint will completely break without refactoring

**Low Risk Areas:**
- Discord verification
- Role management
- Membership granting

---

## Notes

- The `ActiveMember` struct includes `Team` and `Points` in its response. Consider if these should be:
  - Removed from response entirely
  - Fetched via JOIN from new tables
  - Made optional/nullable

- The `User` struct is used in multiple operations (GET, POST, PUT). All operations will need to be updated consistently.

- Consider backward compatibility: If external systems depend on `team` and `points` in user responses, you may need to add JOIN logic to maintain the same API contract.

---

## Points Table Schema Analysis

### **Proposed Schema Issues:**

The proposed `points` table schema has several issues that need to be addressed:

#### **1. Duplicate Foreign Key Constraints** ❌ **CRITICAL ERROR**
```sql
constraint points_userID_fkey foreign KEY ("userID") references users (id) on update RESTRICT on delete set null,
constraint points_userID_fkey1 foreign KEY ("userID") references users (id) on update CASCADE on delete set null
```

**Problem:** You cannot have two foreign key constraints on the same column. PostgreSQL will throw an error.

**Solution:** Choose one constraint. Recommended:
```sql
constraint points_userID_fkey foreign KEY ("userID") references users (id) on update CASCADE on delete SET NULL
```

#### **2. Composite Primary Key Design** ⚠️ **DESIGN DECISION NEEDED**

**Current Schema:**
```sql
constraint points_pkey primary key ("userID", id)
```

**Implications:**
- Allows **multiple rows per user** (point history/audit trail)
- Each point change creates a new row
- To get current points, you need to query the **latest** row for a user

**Questions to Consider:**
1. **Do you need point history?** 
   - If YES: Current schema works, but code needs to get latest row
   - If NO: Change to single row per user: `PRIMARY KEY ("userID")` and remove `id` column

2. **Current API behavior:**
   - Current `/api/users/points` endpoint expects single `points` value per user
   - With history table, you'll need to get `MAX(created_at)` or `ORDER BY created_at DESC LIMIT 1`

#### **3. Column Naming Convention** ⚠️ **CONSISTENCY ISSUE**

**Current:** `"userID"` (camelCase with quotes)
**PostgreSQL Convention:** `user_id` (snake_case)

**Recommendation:** Use `user_id` for consistency with your `users` table and Go struct naming.

#### **4. Updated By Field** ✅ **GOOD FOR AUDIT**

The `updated_by` field is good for tracking who made changes, but:
- Default value `gen_random_uuid()` doesn't make sense - should be `NULL` by default
- Should reference `users.id` (which it does ✅)

---

### **Recommended Schema (Fixed):**

```sql
create table public.points (
  user_id uuid not null,
  points integer null,
  created_at timestamp with time zone not null default now(),
  updated_at timestamp with time zone null default now(),
  updated_by uuid null,  -- Remove default, should be set explicitly
  id uuid not null default gen_random_uuid(),
  
  -- Choose ONE of these primary key options:
  
  -- OPTION A: Single row per user (current points only)
  constraint points_pkey primary key (user_id),
  constraint points_id_key unique (id),
  
  -- OR OPTION B: Multiple rows per user (point history)
  -- constraint points_pkey primary key (user_id, id),
  -- constraint points_id_key unique (id),
  
  constraint points_updated_by_fkey foreign KEY (updated_by) 
    references users (id) on update CASCADE on delete set default,
  constraint points_user_id_fkey foreign KEY (user_id) 
    references users (id) on update CASCADE on delete set null
) TABLESPACE pg_default;
```

---

### **Code Changes Required for Points Table:**

#### **If Using History Table (Multiple Rows Per User):**

**1. Update `getNameAndPointsByColumn` function:**
```go
func getNameAndPointsByColumn(client *supabase.Client, column string, value string) (*UserPoints, error) {
    // Need to JOIN users table and get LATEST points row
    // Query should be:
    // SELECT users.first_name, users.last_name, points.points
    // FROM users
    // JOIN points ON points.user_id = users.id
    // WHERE users.{column} = {value}
    // ORDER BY points.created_at DESC
    // LIMIT 1
    
    // Supabase query would be something like:
    var result []struct {
        FirstName string `json:"first_name"`
        LastName  string `json:"last_name"`
        Points    int    `json:"points"`
    }
    
    // This requires a more complex query - may need raw SQL or multiple queries
}
```

**2. Update `updateUserPoints` function:**
```go
func updateUserPoints(client *supabase.Client, column string, value string, userPoints UserPoints) (int64, error) {
    // Instead of UPDATE, you need to:
    // 1. Get user_id from the column/value
    // 2. INSERT a new row into points table (for history)
    // OR
    // 3. If single row per user: UPDATE the existing row
    
    // For history table (INSERT new row):
    newPointEntry := map[string]interface{}{
        "user_id": userID,  // Get from users table first
        "points": userPoints.Points,
        "updated_by": updatedByUserID,  // From auth context
    }
    _, _, err := client.From(constants.POINTS_TABLE).Insert(newPointEntry, false, "", "", "exact").Execute()
    return count, err
}
```

#### **If Using Single Row Per User (Recommended for Simplicity):**

**1. Update `getNameAndPointsByColumn` function:**
```go
func getNameAndPointsByColumn(client *supabase.Client, column string, value string) (*UserPoints, error) {
    // JOIN query to get user info + points
    // SELECT users.first_name, users.last_name, points.points
    // FROM users
    // LEFT JOIN points ON points.user_id = users.id
    // WHERE users.{column} = {value}
    
    // Supabase doesn't easily support JOINs, so you might need:
    // 1. Query users table first to get user_id
    // 2. Query points table with user_id
    // 3. Combine results
}
```

**2. Update `updateUserPoints` function:**
```go
func updateUserPoints(client *supabase.Client, column string, value string, userPoints UserPoints) (int64, error) {
    // 1. Get user_id from users table using column/value
    // 2. Check if points row exists for user_id
    // 3. If exists: UPDATE points SET points = ?, updated_at = now(), updated_by = ?
    // 4. If not exists: INSERT new row
    
    // This requires checking existence first, then upsert logic
}
```

---

### **Recommendation:**

**For your current use case, I recommend:**

1. **Single row per user** (simpler, matches current API behavior)
2. **Fix the schema issues** (remove duplicate FK, use snake_case)
3. **Add helper function** to get/upsert points (since Supabase doesn't support UPSERT easily)

**Alternative:** If you need point history for audit purposes, consider:
- Keep current points in a `user_points` table (single row per user)
- Create separate `points_history` table for audit trail
- This gives you both current value and history

---

### **Updated Migration Impact:**

With the points table schema, the `/api/users/points/index.go` refactor becomes **MORE COMPLEX** because:

1. **JOIN required** to get user names + points
2. **Latest row logic** needed if using history table
3. **Upsert logic** needed for updates (check existence, then insert/update)
4. **Two-step queries** may be needed (Supabase client limitations with JOINs)

**Estimated Complexity:** High → Very High (depending on history vs. single row design)
