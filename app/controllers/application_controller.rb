# frozen_string_literal: true

class ApplicationController < ActionController::Base
  include GraphqlRunnable
  # include SignInTokenAuthenticatable
end
