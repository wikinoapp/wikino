# typed: false
# frozen_string_literal: true

class InitialSetup < ActiveRecord::Migration[7.1]
  def change
    enable_extension(:citext) unless extension_enabled?("citext")
    enable_extension(:pgcrypto) unless extension_enabled?("pgcrypto")

    # Add generate_ulid() which generates ULIDs
    # ref: https://blog.daveallie.com/ulid-primary-keys/
    connection.execute <<~EOS
      CREATE OR REPLACE FUNCTION generate_ulid() RETURNS uuid
      AS $$
        SELECT (lpad(to_hex(floor(extract(epoch FROM clock_timestamp()) * 1000)::bigint), 12, '0') || encode(gen_random_bytes(10), 'hex'))::uuid;
      $$ LANGUAGE SQL;
    EOS
  end
end
