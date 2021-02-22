# frozen_string_literal: true

module Internal
  class ApplicationController < ActionController::API
    include ActionController::HttpAuthentication::Token::ControllerMethods

    private

    def authenticate_with_access_token
      @current_user = authenticate_with_http_token do |token|
        User.only_kept.find_by(access_token: token)
      end
    end

    def current_user
      @current_user
    end
  end
end
