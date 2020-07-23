# frozen_string_literal: true

class ApplicationController < ActionController::Base
  helper_method :current_team, :current_team_member

  private

  def graphql_client
    @graphql_client ||= Nonoto::Graphql::InternalClient.new(
      viewer: current_team_member
    )
  end

  def current_team
    return unless user_signed_in?
    return current_user.teams.first unless params[:team_id]

    current_user.teams.find(params[:team_id])
  end

  def current_team_member
    return unless user_signed_in?

    current_user.team_members.find_by!(team: current_team)
  end
end
