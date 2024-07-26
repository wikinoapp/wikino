# typed: true
# frozen_string_literal: true

class Notes::Info::ShowController < ApplicationController
  include Authenticatable

  before_action :authenticate_user

  sig { returns(T.untyped) }
  def call
    # @note = T.must(current_user).notes.find(params[:note_id])
  end
end
