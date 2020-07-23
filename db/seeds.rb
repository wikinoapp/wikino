# frozen_string_literal: true

puts "Creating users..."
user_1 = User.where(email: "user_1@example.com").first_or_create!(password: "useruser")
user_1.confirm

puts "Creating teams..."
team_1 = Team.where(teamname: "team-1").first_or_create!(name: "Team 1")

puts "Creating team_members..."
team_member_1 = team_1.team_members.where(user: user_1).first_or_create!(name: "User 1")

puts "Creating projects..."
project_1 = team_1.projects.where(name: "Project 1").create!

puts "Creating notes..."
note_1 = team_1.notes.where(number: 1).first_or_create!(team_member: team_member_1, project: project_1, title: "Note 1", body: "This is the Note 1.")
note_2 = team_1.notes.where(number: 2).first_or_create!(team_member: team_member_1, project: project_1, title: "Note 2", body: "This is the Note 2.")

puts "Creating references..."
note_1.referencing_references.where(referencing_note: note_2).first_or_create!

puts "Creating tags..."
tag_1 = project_1.tags.where(name: "Tag 1").first_or_create!
tag_2 = project_1.tags.where(name: "Tag 2").first_or_create!

puts "Creating taggings..."
note_1.taggings.where(tag: tag_1).first_or_create!
note_1.taggings.where(tag: tag_2).first_or_create!
