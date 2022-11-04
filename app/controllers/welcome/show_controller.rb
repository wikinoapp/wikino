# typed: strict
# frozen_string_literal: true

class Welcome::ShowController < ApplicationController
  extend T::Sig

  include Authenticatable

  sig { returns(T.untyped) }
  def call
  end
end
