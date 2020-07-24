# frozen_string_literal: true
# == Schema Information
#
# Table name: references
#
#  id                  :uuid             not null, primary key
#  created_at          :datetime         not null
#  updated_at          :datetime         not null
#  note_id             :uuid             not null
#  referencing_note_id :uuid             not null
#
# Indexes
#
#  index_references_on_created_at                       (created_at)
#  index_references_on_note_id                          (note_id)
#  index_references_on_note_id_and_referencing_note_id  (note_id,referencing_note_id) UNIQUE
#  index_references_on_referencing_note_id              (referencing_note_id)
#
# Foreign Keys
#
#  fk_rails_...  (note_id => notes.id)
#  fk_rails_...  (referencing_note_id => notes.id)
#
class Reference < ApplicationRecord
  belongs_to :note
  belongs_to :referencing_note, class_name: "Note"
end
