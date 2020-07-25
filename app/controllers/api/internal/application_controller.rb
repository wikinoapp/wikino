# frozen_string_literal: true

module Api
  module Internal
    class ApplicationController < ActionController::Base
      include GraphqlRunnable

      layout false
    end
  end
end
