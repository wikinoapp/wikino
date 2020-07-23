# frozen_string_literal: true

class WelcomeController < ApplicationController
  def show
    redirect_to note_list_path(team_id: current_team.id) if user_signed_in?
  end
end
