# frozen_string_literal: true
# == Schema Information
#
# Table name: taggings
#
#  id         :bigint           not null, primary key
#  created_at :datetime         not null
#  updated_at :datetime         not null
#  note_id    :bigint           not null
#  tag_id     :bigint           not null
#
# Indexes
#
#  index_taggings_on_created_at          (created_at)
#  index_taggings_on_note_id             (note_id)
#  index_taggings_on_note_id_and_tag_id  (note_id,tag_id) UNIQUE
#  index_taggings_on_tag_id              (tag_id)
#
# Foreign Keys
#
#  fk_rails_...  (note_id => notes.id)
#  fk_rails_...  (tag_id => tags.id)
#
class Tagging < ApplicationRecord
  belongs_to :note
  belongs_to :tag
end
