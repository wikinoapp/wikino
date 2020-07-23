# frozen_string_literal: true
# == Schema Information
#
# Table name: notes
#
#  id             :bigint           not null, primary key
#  body           :text             default(""), not null
#  number         :bigint           not null
#  title          :string           default(""), not null
#  created_at     :datetime         not null
#  updated_at     :datetime         not null
#  project_id     :bigint           not null
#  team_id        :bigint           not null
#  team_member_id :bigint           not null
#
# Indexes
#
#  index_notes_on_created_at            (created_at)
#  index_notes_on_project_id            (project_id)
#  index_notes_on_project_id_and_title  (project_id,title) UNIQUE
#  index_notes_on_team_id               (team_id)
#  index_notes_on_team_id_and_number    (team_id,number) UNIQUE
#  index_notes_on_team_member_id        (team_member_id)
#  index_notes_on_updated_at            (updated_at)
#
# Foreign Keys
#
#  fk_rails_...  (project_id => projects.id)
#  fk_rails_...  (team_id => teams.id)
#  fk_rails_...  (team_member_id => team_members.id)
#
class Note < ApplicationRecord
  belongs_to :team
  belongs_to :project
  belongs_to :team_member

  has_many :referenced_references, class_name: "Reference", dependent: :destroy, foreign_key: :referencing_note_id
  has_many :referenced_notes, class_name: "Note", source: :note, through: :referenced_references
  has_many :referencing_references, class_name: "Reference", dependent: :destroy, foreign_key: :note_id
  has_many :referencing_notes, class_name: "Note", through: :referencing_references
  has_many :taggings, dependent: :destroy
  has_many :tags, through: :taggings
end
