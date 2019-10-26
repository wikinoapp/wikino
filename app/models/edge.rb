# frozen_string_literal: true

# == Schema Information
#
# Table name: edges
#
#  id             :uuid             not null, primary key
#  created_at     :datetime         not null
#  updated_at     :datetime         not null
#  note_id        :uuid             not null
#  target_note_id :uuid             not null
#
# Indexes
#
#  index_edges_on_note_id                     (note_id)
#  index_edges_on_note_id_and_target_note_id  (note_id,target_note_id) UNIQUE
#  index_edges_on_target_note_id              (target_note_id)
#
# Foreign Keys
#
#  fk_rails_...  (note_id => notes.id)
#  fk_rails_...  (target_note_id => notes.id)
#

class Edge < ApplicationRecord
end
