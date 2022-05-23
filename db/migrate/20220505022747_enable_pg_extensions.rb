# typed: false
# frozen_string_literal: true

class EnablePgExtensions < ActiveRecord::Migration[7.0]
  def change
    enable_extension(:citext) unless extension_enabled?("citext")
    enable_extension(:pgcrypto) unless extension_enabled?("pgcrypto")
  end
end
