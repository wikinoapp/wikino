# frozen_string_literal: true
# == Schema Information
#
# Table name: notes
#
#  id         :bigint           not null, primary key
#  body       :text             default(""), not null
#  title      :citext           default(""), not null
#  created_at :datetime         not null
#  updated_at :datetime         not null
#  user_id    :bigint           not null
#
# Indexes
#
#  index_notes_on_created_at         (created_at)
#  index_notes_on_updated_at         (updated_at)
#  index_notes_on_user_id            (user_id)
#  index_notes_on_user_id_and_title  (user_id,title) UNIQUE
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
  has_many :taggings, dependent: :destroy
  has_many :tags, through: :taggings
end
