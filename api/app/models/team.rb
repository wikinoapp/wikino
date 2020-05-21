# frozen_string_literal: true

# == Schema Information
#
# Table name: teams
#
#  id         :uuid             not null, primary key
#  deleted_at :datetime
#  name       :string           default(""), not null
#  teamname   :citext           not null
#  created_at :datetime         not null
#  updated_at :datetime         not null
#
# Indexes
#
#  index_teams_on_deleted_at  (deleted_at)
#  index_teams_on_teamname    (teamname) UNIQUE
#
class Team < ApplicationRecord
end
