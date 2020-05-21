# frozen_string_literal: true

# == Schema Information
#
# Table name: participants
#
#  id             :uuid             not null, primary key
#  created_at     :datetime         not null
#  updated_at     :datetime         not null
#  note_id        :uuid             not null
#  team_member_id :uuid             not null
#
# Indexes
#
#  index_participants_on_note_id                     (note_id)
#  index_participants_on_note_id_and_team_member_id  (note_id,team_member_id) UNIQUE
#  index_participants_on_team_member_id              (team_member_id)
#
# Foreign Keys
#
#  fk_rails_...  (note_id => notes.id)
#  fk_rails_...  (team_member_id => team_members.id)
#
class Participant < ApplicationRecord
end
