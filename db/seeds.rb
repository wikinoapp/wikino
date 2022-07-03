# typed: strict
# frozen_string_literal: true

puts "Creating users..."
user_1 = User.where(email: "user_1@example.com").first_or_create!(password: "useruser")
user_1.confirm

user_2 = User.where(email: "user_2@example.com").first_or_create!(password: "useruser")
user_2.confirm

puts "Creating notes..."
200.times do |i|
  user_1.notes.where(title: "User 1 Note ##{i}").first_or_create!(body: "This is Note #{i}.", body_html: "This is Note #{i}.", modified_at: Time.zone.now)
end

200.times do |i|
  user_2.notes.where(title: "User 2 Note ##{i}").first_or_create!(body: "This is Note #{i}.", body_html: "This is Note #{i}.", modified_at: Time.zone.now)
end
