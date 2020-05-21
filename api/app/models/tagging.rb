# frozen_string_literal: true

# == Schema Information
#
# Table name: taggings
#
#  id         :uuid             not null, primary key
#  created_at :datetime         not null
#  updated_at :datetime         not null
#  note_id    :uuid             not null
#  project_id :uuid             not null
#  tag_id     :uuid             not null
#  team_id    :uuid             not null
#
# Indexes
#
#  index_taggings_on_note_id             (note_id)
#  index_taggings_on_note_id_and_tag_id  (note_id,tag_id) UNIQUE
#  index_taggings_on_project_id          (project_id)
#  index_taggings_on_tag_id              (tag_id)
#  index_taggings_on_team_id             (team_id)
#
# Foreign Keys
#
#  fk_rails_...  (note_id => notes.id)
#  fk_rails_...  (project_id => projects.id)
#  fk_rails_...  (tag_id => tags.id)
#  fk_rails_...  (team_id => teams.id)
#
class Tagging < ApplicationRecord
end
