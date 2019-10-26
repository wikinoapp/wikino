# frozen_string_literal: true

# == Schema Information
#
# Table name: notes
#
#  id         :uuid             not null, primary key
#  body       :text
#  number     :integer          not null
#  created_at :datetime         not null
#  updated_at :datetime         not null
#  creator_id :uuid             not null
#  project_id :uuid             not null
#  team_id    :uuid             not null
#
# Indexes
#
#  index_notes_on_creator_id          (creator_id)
#  index_notes_on_project_id          (project_id)
#  index_notes_on_team_id             (team_id)
#  index_notes_on_team_id_and_number  (team_id,number) UNIQUE
#
# Foreign Keys
#
#  fk_rails_...  (creator_id => users.id)
#  fk_rails_...  (project_id => projects.id)
#  fk_rails_...  (team_id => teams.id)
#

class Note < ApplicationRecord
end
