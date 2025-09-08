# Issue Management Implementation Summary

## Quick Decision Points for Review

### 1. Issue Categories
**Proposed Main Categories:**
- **Quality Issues** - Defective work, material defects, workmanship
- **Safety Issues** - Hazards, incidents, PPE violations  
- **Punch List** - Final completion items
- **Inspections** - Code compliance, pre-pour, rough-in
- **Warranty Items** - 30/90/365 day items
- **Environmental** - Erosion, waste, noise, dust

**Questions:**
- Do these categories match your needs?
- Any additional categories required?

### 2. Issue Numbering Format
**Proposed:** `[PROJECT_CODE]-[CATEGORY]-[SEQUENCE]`
Example: `TWR-A-QC-0145` (Tower A, Quality Control, Issue #145)

**Alternative:** Simple sequential `ISS-000001`

### 3. Priority Levels
**Current API:** critical, high, medium, low, planned
**Proposed:** emergency, critical, high, medium, low, planned

### 4. Severity Levels  
**Current API:** blocking, major, minor, cosmetic
**Keep as-is or add:** critical?

### 5. Status Workflow

**Standard Issues:**
```
Draft → Open → Acknowledged → In Progress → Ready for Inspection → Verified → Closed
```

**Simplified Option:**
```
Open → In Progress → Resolved → Closed
```

### 6. Location Tracking Options

**Option A: Full Construction Tracking**
- Drawing coordinates (x,y)
- Building/Floor/Room
- Grid lines (construction grids)
- GPS coordinates
- Area names

**Option B: Simplified**
- Location description (text)
- Building/Floor/Room
- Optional coordinates

### 7. Key Features Priority

**Must Have (Phase 1):**
- Basic CRUD operations
- Photo attachments
- Assignment to users/companies
- Email notifications
- Status workflow

**Should Have (Phase 2):**
- Templates system
- Auto-assignment by trade
- Cost tracking
- Bulk operations
- Distribution lists

**Nice to Have (Phase 3):**
- Drawing pin/markup
- Ball-in-court tracking
- Checklists
- Issue linking
- Root cause analysis

### 8. Photo Requirements
- **Before photo** - Required for all issues?
- **After photo** - Required for closure?
- **Progress photos** - Optional?

### 9. Assignment Logic

**Option A: Complex**
- Auto-assign based on category→trade→contractor matrix
- Escalation chains
- Workload balancing

**Option B: Simple**
- Manual assignment only
- Default to project manager

### 10. Notification Triggers

**Essential:**
- New issue → Assigned party
- Status change → Creator + Assigned
- Issue closed → All stakeholders

**Additional:**
- Due date reminder (1 day before)
- Overdue escalation
- Comment added

## Database Changes Summary

### New Tables Required:
1. `issue_types` - Category definitions
2. `issue_templates` - Reusable templates
3. `issue_attachments` - Enhanced photo/file tracking
4. `issue_history` - Full audit trail
5. `issue_checklists` - Optional inspection items
6. `trade_responsibility_matrix` - Auto-assignment mapping

### Updates to Existing `issues` Table:
1. Add template support fields
2. Add structured location fields  
3. Add trade/discipline fields
4. Add distribution list
5. Update constraints for new priority/severity values

## API Endpoints Summary

### Core Operations:
```
POST   /projects/{id}/issues          - Create issue
GET    /projects/{id}/issues          - List project issues
GET    /issues/{id}                   - Get issue details
PUT    /issues/{id}                   - Update issue
DELETE /issues/{id}                   - Soft delete
PATCH  /issues/{id}/status           - Update status only
POST   /issues/{id}/attachments      - Upload photo/file
POST   /issues/{id}/comments         - Add comment
```

### Templates:
```
GET    /issue-templates               - List templates
POST   /issue-templates               - Create template
POST   /templates/{id}/create-issue   - Create from template
```

## Implementation Approach

### Week 1 - Foundation
- [ ] Create database migrations
- [ ] Implement basic Issue model
- [ ] Create Lambda handler
- [ ] Basic CRUD operations
- [ ] Simple photo upload

### Week 2 - Core Features  
- [ ] Template system
- [ ] Status workflow
- [ ] Email notifications
- [ ] Assignment logic
- [ ] Comments/history

### Week 3 - Advanced
- [ ] Bulk operations
- [ ] Cost tracking
- [ ] Auto-assignment
- [ ] Distribution lists
- [ ] Reporting endpoints

## Questions to Answer Before Starting:

1. **Complexity Level:** Full construction features or start simple?
2. **Issue Numbering:** Which format do you prefer?
3. **Categories:** Use proposed categories or customize?
4. **Templates:** Required from day 1 or add later?
5. **Auto-Assignment:** Implement complex logic or keep manual?
6. **Location Tracking:** Full spatial support or basic text?
7. **Workflow:** Complex status flow or simplified?
8. **Photos:** Require before/after or make optional?
9. **Distribution Lists:** Email only or add SMS?
10. **Priority:** Which features are must-have for MVP?

## Recommended MVP Scope

Start with:
1. Basic issue CRUD
2. Simple categories (no templates initially)
3. Photo upload (before/after)
4. Manual assignment
5. Basic email notifications
6. Simple status workflow (Open→In Progress→Closed)
7. Text-based location description

Then add:
1. Templates
2. Auto-assignment
3. Cost tracking
4. Advanced location tracking
5. Distribution lists

This approach gets you operational quickly while leaving room for enhancement based on user feedback.