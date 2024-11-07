FealtyX API is a RESTful service built in Go for managing student records, complete with CRUD (Create, Read, Update, Delete) functionality. It also integrates with Ollama to generate AI-based summaries for each student.

Features
Student Management: Create, retrieve, update, and delete student records.
AI-Powered Summaries: Generate summaries for student profiles using Ollama.
Public Exposure via Serveo: Expose the API publicly for testing and demos.

API Endpoints
Method	Endpoint	Description
POST	/students	Create a new student
GET	/students	Retrieve all students
GET	/students/{id}	Retrieve a student by ID
PUT	/students/{id}	Update a student by ID
DELETE	/students/{id}	Delete a student by ID
GET	/students/{id}/summary	Get AI-based summary of a student

**IMPORTANT**
I have made this public on "https://muftiusmanapiforfealtyx.serveo.net/" using Serveo. Serveo allows to create a secure tunnel, forwarding traffic from a public URL to my local server. It provides temporary hosting 
via SSH tunneling, therefore it requires the SSH session TO REMAIN OPEN. The API server relies on my local resources, so it will stop if I close the terminal or shut down your machine or unknowingly terminate.
