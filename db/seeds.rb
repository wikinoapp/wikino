# frozen_string_literal: true

puts "Creating users..."
user_1 = User.where(email: "user_1@example.com").first_or_create!(password: "useruser")
user_1.confirm

user_2 = User.where(email: "user_2@example.com").first_or_create!(password: "useruser")
user_2.confirm


puts "Creating notes..."
note_1 = user_1.notes.where(title: "Note 1").first_or_create!(body: "This is a first note.", body_html: "This is a first note.")
note_2 = user_1.notes.where(title: "Note 2").first_or_create!(body: "This is a second note.", body_html: "This is a second note.")


puts "Creating links..."
note_1.referencing_links.where(target_note: note_2).first_or_create!
