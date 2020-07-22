# frozen_string_literal: true

puts "Creating users..."
user_1 = User.where(email: "user_1@example.com").first_or_create!(password: "user1user1")

puts "Creating notes..."
user_1_note_1 = user_1.notes.where(number: 1).first_or_create!(title: "Note 1", body: "Note 1")
user_1_note_2 = user_1.notes.where(number: 2).first_or_create!(title: "Note 2", body: "Note 2")

puts "Creating references..."
user_1_note_1.referencing_references.where(referencing_note: user_1_note_2).first_or_create!

puts "Creating tags..."
user_1_tag_1 = user_1.tags.where(name: "Tag 1").first_or_create!
user_1_tag_2 = user_1.tags.where(name: "Tag 2").first_or_create!

puts "Creating taggings..."
user_1_note_1.taggings.where(tag: user_1_tag_1).first_or_create!
user_1_note_1.taggings.where(tag: user_1_tag_2).first_or_create!
