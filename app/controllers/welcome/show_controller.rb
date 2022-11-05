# typed: strict
# frozen_string_literal: true

class Welcome::ShowController < ApplicationController
  extend T::Sig

  include Authenticatable

  sig { returns(T.untyped) }
  def call
    if user_signed_in?
      return render :call_signed_in
    end

    render :call
  end
end
