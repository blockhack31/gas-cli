# Use Cases

## Freelancers & Consultants

```bash
gascli add client-a contact@client-a.com "Your Name" CLIENT_A_GPG
gascli add client-b contact@client-b.com "Your Name" CLIENT_B_GPG

gascli auto ~/clients/client-a client-a
gascli auto ~/clients/client-b client-b
```

Commits automatically signed with correct identity per project.

## Work/Personal Separation

```bash
gascli add work jane@company.com "Jane Doe" WORK_GPG
gascli add personal jane@example.com "Jane Smith" PERSONAL_GPG

gascli auto ~/work work
gascli auto ~/personal personal
gascli auto ~/opensource personal
```

## Development Teams

Share standardized profiles:

```bash
# Team lead
gascli export > team-profiles.json

# Team members
gascli import team-profiles.json
gascli auto ~/projects/client-a client-a-profile
```

## Multiple Company Roles

```bash
gascli add work jane@company.com "Jane Doe"
gascli add-email work jane.contractor@company.com
gascli add-email work j.doe@consulting.com

# Switch between roles
gascli switch work jane.contractor@company.com
```

## Open Source Contributors

```bash
gascli add personal dev@example.com "Your Name"
gascli add work-oss dev@company.com "Your Name (Company)"

gascli auto ~/oss personal
gascli auto ~/work-oss work-oss
```

## Educational Institutions

```bash
gascli add student student.id@university.edu "Student Name"
gascli add research prof@university.edu "Dr. Name" RESEARCH_GPG

gascli auto ~/courses student
gascli auto ~/research research
```

## Compliance Requirements

Organizations requiring GPG signing:

```bash
gascli add company email@corp.com "Name" COMPANY_GPG_KEY
gascli auto ~/corp-repos company
```

All commits automatically signed, audit trail maintained.
