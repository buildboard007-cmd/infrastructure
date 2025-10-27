# Archived Documentation

**Last Updated:** 2025-10-25

These documents are historical planning and analysis documents created during development.
They are preserved for historical reference but are **OUTDATED** and should not be used for current development.

---

## 📦 Archived Documents

### Planning Documents (Implementation Complete)

#### 1. EXISTING-IAM-ARCHITECTURE-AND-PROJECT-ACCESS-PLAN.md (17K)
**Created:** Planning phase
**Status:** Implementation completed
**Superseded by:** APPLICATION-ARCHITECTURE.md
**Why archived:** References old table structure (`org_user_roles`, `location_user_roles`, `project_user_roles`) that have been migrated to unified `user_assignments` table.

#### 2. PROJECT-ACCESS-IMPLEMENTATION-FINAL.md (12K)
**Created:** Planning phase
**Status:** Implementation completed
**Superseded by:** APPLICATION-ARCHITECTURE.md
**Why archived:** Implementation plan - work is done and documented in main architecture doc.

#### 3. project-access-control-design.md (11K)
**Created:** Design phase
**Status:** Implementation completed
**Superseded by:** APPLICATION-ARCHITECTURE.md
**Why archived:** Design document - final implementation documented elsewhere.

---

### Analysis Documents (Superseded)

#### 4. DATABASE-ARCHITECTURE-ANALYSIS.md (11K)
**Created:** Before October 2025 migration
**Status:** Analysis complete, migration done
**Superseded by:** APPLICATION-ARCHITECTURE.md
**Why archived:** Analyzed tables as "transitional" - migration is now complete. Tables mentioned have been dropped or migrated.

#### 5. issue-management-missing-features.md (15K)
**Created:** Gap analysis phase
**Status:** Likely outdated
**Superseded by:** Current implementation
**Why archived:** Gap analysis document - features may have been implemented or deprioritized.

#### 6. issue-attachment-validation-gaps.md (2.9K)
**Created:** Validation analysis phase
**Status:** Addressed by centralized attachment system
**Superseded by:** USER_GUIDE_ENTITY_ATTACHMENTS.md
**Why archived:** Validation gaps addressed by unified attachment management system.

---

### Architecture Documents (Consolidated)

#### 7. multi-api-architecture.md (7.4K)
**Created:** Architecture documentation
**Status:** Consolidated
**Superseded by:** APPLICATION-ARCHITECTURE.md "API Architecture" section
**Why archived:** Content merged into comprehensive architecture document.

#### 8. postman-collection-migration.md (5.5K)
**Created:** Postman collection management guide
**Status:** Historical
**Superseded by:** Current Postman collections in `/postman/`
**Why archived:** Migration guide no longer needed, collections are up-to-date.

---

## ⚠️ Important Notes

### DO NOT Use These Documents For:
- ❌ Current development work
- ❌ Understanding current architecture
- ❌ API reference
- ❌ Database schema reference

### These Documents Reference DROPPED Tables:
- `iam.org_user_roles` (dropped October 2025)
- `iam.location_user_roles` (dropped October 2025)
- `iam.user_location_access` (dropped October 2025)
- `project.project_user_roles` (dropped October 2025)
- `project.project_managers` (dropped October 2025)

**All these tables have been replaced by:** `iam.user_assignments`

---

## ✅ Use These Active Documents Instead

**For complete system context:**
→ Read [../APPLICATION-ARCHITECTURE.md](../APPLICATION-ARCHITECTURE.md)

**For quick navigation:**
→ Read [../DOCUMENTATION-INDEX.md](../DOCUMENTATION-INDEX.md)

**For migration history:**
→ Read [../CHANGES-SUMMARY.md](../CHANGES-SUMMARY.md)

**For assignment system details:**
→ Read [../assignment-architecture.md](../assignment-architecture.md)

---

## 📚 Historical Value

These documents are preserved because they:
- Document the decision-making process
- Show the evolution of the architecture
- Provide context for why certain decisions were made
- Serve as reference for understanding the migration process

**Use them only for:** Understanding historical context and architectural decisions.

---

**Archive Created:** 2025-10-25
**Total Documents Archived:** 8
**Total Size:** ~87K