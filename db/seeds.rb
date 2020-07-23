# frozen_string_literal: true

puts "Creating users..."
user_1 = User.where(email: "user_1@example.com").first_or_create!(password: "useruser")
user_1.confirm

puts "Creating notes..."
note_1 = user_1.notes.where(title: "Note 1").first_or_create!(body: "This is the Note 1.")
note_2 = user_1.notes.where(title: "Note 2").first_or_create!(body: "This is the Note 2.")

puts "Creating references..."
note_1.referencing_references.where(referencing_note: note_2).first_or_create!

puts "Creating tags..."
tag_1 = user_1.tags.where(name: "Tag 1").first_or_create!
tag_2 = user_1.tags.where(name: "Tag 2").first_or_create!

puts "Creating taggings..."
note_1.taggings.where(tag: tag_1).first_or_create!
note_1.taggings.where(tag: tag_2).first_or_create!
