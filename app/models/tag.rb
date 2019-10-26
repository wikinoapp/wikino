# frozen_string_literal: true

# == Schema Information
#
# Table name: tags
#
#  id         :uuid             not null, primary key
#  name       :citext           not null
#  created_at :datetime         not null
#  updated_at :datetime         not null
#  project_id :uuid             not null
#  team_id    :uuid             not null
#
# Indexes
#
#  index_tags_on_project_id           (project_id)
#  index_tags_on_project_id_and_name  (project_id,name) UNIQUE
#  index_tags_on_team_id              (team_id)
#
# Foreign Keys
#
#  fk_rails_...  (project_id => projects.id)
#  fk_rails_...  (team_id => teams.id)
#

class Tag < ApplicationRecord
end
