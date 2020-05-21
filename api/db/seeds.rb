# frozen_string_literal: true

ActiveRecord::Base.transaction do
  user_1 = User.create!(
    email: "user_1@example.com",
    username: "user_1",
    name: "User 1"
  )

  user_1.oauth_providers.create!(
    name: :google,
    token: "user_1-token",
    token_expires_at: 1111111111,
    uid: "user_1-uid"
  )

  team_1 = Team.create!(
    teamname: "team_1",
    name: "Team 1"
  )

  team_member_1 = team_1.team_members.create!(
    user: user_1,
    username: user_1.username,
    name: user_1.name
  )

  project_1 = team_1.projects.create!(
    projectname: "general",
    visibility: :private
  )

  project_1.project_members.create!(
    team_member: team_member_1
  )
end
