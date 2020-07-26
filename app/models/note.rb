# frozen_string_literal: true
# == Schema Information
#
# Table name: notes
#
#  id         :uuid             not null, primary key
#  body       :text             default(""), not null
#  name       :citext           default(""), not null
#  created_at :datetime         not null
#  updated_at :datetime         not null
#  user_id    :uuid             not null
#
# Indexes
#
#  index_notes_on_created_at        (created_at)
#  index_notes_on_updated_at        (updated_at)
#  index_notes_on_user_id           (user_id)
#  index_notes_on_user_id_and_name  (user_id,name) UNIQUE
#
# Foreign Keys
#
#  fk_rails_...  (user_id => users.id)
#
class Note < ApplicationRecord
  belongs_to :user

  has_many :referenced_references, class_name: "Reference", dependent: :destroy, foreign_key: :referencing_note_id
  has_many :referenced_notes, class_name: "Note", source: :note, through: :referenced_references
  has_many :referencing_references, class_name: "Reference", dependent: :destroy, foreign_key: :note_id
  has_many :referencing_notes, class_name: "Note", through: :referencing_references

  validates :name, presence: true
  validates :body, presence: true

  def set_name!
    self.name = body.split("\n").first&.strip
  end
end
