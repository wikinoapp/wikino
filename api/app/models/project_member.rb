# frozen_string_literal: true

# == Schema Information
#
# Table name: project_members
#
#  id             :uuid             not null, primary key
#  created_at     :datetime         not null
#  updated_at     :datetime         not null
#  project_id     :uuid             not null
#  team_member_id :uuid             not null
#
# Indexes
#
#  index_project_members_on_project_id                     (project_id)
#  index_project_members_on_project_id_and_team_member_id  (project_id,team_member_id) UNIQUE
#  index_project_members_on_team_member_id                 (team_member_id)
#
# Foreign Keys
#
#  fk_rails_...  (project_id => projects.id)
#  fk_rails_...  (team_member_id => team_members.id)
#
class ProjectMember < ApplicationRecord
end
