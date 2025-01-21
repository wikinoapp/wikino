# frozen_string_literal: true

User.find_each do |user|
  user.space_memberships.create!(space: user.space, role: user.role, joined_at: user.created_at)
end
