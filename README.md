# Share Time With Me

## Overview

This project provides a simple platform for friends to schedule meetings with ease. It allows users to select their available days for meetings during the current or next week, which are then processed and stored efficiently in a MongoDB collection hosted on MongoDB Atlas. The backend is built with Golang, while the frontend is powered by Next.js for a seamless user experience.

## Features

- **User-Friendly Meeting Scheduling**: Users can select their availability for the current and/or upcoming week.
- **Efficient Storage**: The selected availability is converted into binary representation for the days and then stored as a decimal value in a MongoDB Atlas database.
- **Backend in Go**: The server-side is developed using Golang for efficient performance and simplicity.
- **Frontend in Next.js**: The client-side is built with Next.js to deliver a fast, interactive UI.
- **MongoDB Atlas for Persistence**: All user data is stored in a cloud-hosted MongoDB database (MongoDB Atlas).

## Technical Stack

- **Frontend**: [Next.js](https://nextjs.org/)
- **Backend**: [Go (Golang)](https://golang.org/)
- **Database**: [MongoDB Atlas](https://www.mongodb.com/cloud/atlas)

## Data Model

- **Availability Storage**: Each user's availability is represented as a binary string where each bit corresponds to a day of the week (0 for unavailable, 1 for available).
  - For example, a user's availability for Monday and Wednesday in a week would look like `1010000` (where the first bit represents Sunday, the second Monday, and so on).
  - This binary string is converted to its decimal equivalent and stored in MongoDB for efficient storage and retrieval.
