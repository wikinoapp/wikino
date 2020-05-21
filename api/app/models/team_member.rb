# frozen_string_literal: true

# == Schema Information
#
# Table name: team_members
#
#  id         :uuid             not null, primary key
#  deleted_at :datetime
#  name       :string           default(""), not null
#  username   :citext           not null
#  created_at :datetime         not null
#  updated_at :datetime         not null
#  team_id    :uuid             not null
#  user_id    :uuid             not null
#
# Indexes
#
#  index_team_members_on_deleted_at            (deleted_at)
#  index_team_members_on_team_id               (team_id)
#  index_team_members_on_team_id_and_user_id   (team_id,user_id) UNIQUE
#  index_team_members_on_team_id_and_username  (team_id,username) UNIQUE
#  index_team_members_on_user_id               (user_id)
#
# Foreign Keys
#
#  fk_rails_...  (team_id => teams.id)
#  fk_rails_...  (user_id => users.id)
#
class TeamMember < ApplicationRecord
end
