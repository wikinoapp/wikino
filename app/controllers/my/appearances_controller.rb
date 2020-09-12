# frozen_string_literal: true

module My
  class AppearancesController < ApplicationController
    before_action :authenticate_user!

    def show; end
  end
end
