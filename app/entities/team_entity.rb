# frozen_string_literal: true

class TeamEntity < ApplicationEntity
  attribute? :teamname, Types::String

  def self.from_node(team_node)
    attrs = {}

    if teamname = team_node["teamname"]
      attrs[:teamname] = teamname
    end

    new attrs
  end
end
