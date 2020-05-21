# frozen_string_literal: true

# == Schema Information
#
# Table name: projects
#
#  id          :uuid             not null, primary key
#  deleted_at  :datetime
#  name        :string           default(""), not null
#  projectname :string           not null
#  visibility  :string           not null
#  created_at  :datetime         not null
#  updated_at  :datetime         not null
#  team_id     :uuid             not null
#
# Indexes
#
#  index_projects_on_deleted_at               (deleted_at)
#  index_projects_on_team_id                  (team_id)
#  index_projects_on_team_id_and_projectname  (team_id,projectname) UNIQUE
#
# Foreign Keys
#
#  fk_rails_...  (team_id => teams.id)
#
class Project < ApplicationRecord
end
