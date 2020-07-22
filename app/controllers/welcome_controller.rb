# frozen_string_literal: true

class WelcomeController < ApplicationController
  def show
    redirect_to home_path if user_signed_in?
  end
end
